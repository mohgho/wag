package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/NHAS/wag/internal/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Device struct {
	Version      int
	Address      string
	Publickey    string
	Username     string
	PresharedKey string
	Endpoint     *net.UDPAddr
	Attempts     int
	Active       bool
}

func stringToUDPaddr(address string) (r *net.UDPAddr) {
	parts := strings.Split(address, ":")
	if len(parts) < 2 {
		return nil
	}

	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return nil
	}

	r = &net.UDPAddr{
		IP:   net.ParseIP(utils.GetIP(address)),
		Port: port,
	}

	return
}

func UpdateDeviceEndpoint(address string, endpoint *net.UDPAddr) error {

	realKey, err := etcd.Get(context.Background(), "deviceref-"+address)
	if err != nil {
		return err
	}

	if realKey.Count == 0 {
		return errors.New("device was not found")
	}

	return doSafeUpdate(context.Background(), string(realKey.Kvs[0].Value), false, func(gr *clientv3.GetResponse) (string, bool, error) {
		if len(gr.Kvs) != 1 {
			return "", false, errors.New("user device has multiple keys")
		}

		var device Device
		err := json.Unmarshal(gr.Kvs[0].Value, &device)
		if err != nil {
			return "", false, err
		}

		device.Endpoint = endpoint

		b, _ := json.Marshal(device)

		return string(b), false, err
	})
}

func GetDevice(username, id string) (device Device, err error) {

	response, err := etcd.Get(context.Background(), deviceKey(username, id))
	if err != nil {
		return Device{}, err
	}

	if response.Count == 0 {
		return Device{}, errors.New("device was not found")
	}

	if len(response.Kvs) != 1 {
		return Device{}, errors.New("user device has multiple keys")
	}

	err = json.Unmarshal(response.Kvs[0].Value, &device)

	return
}

func SetDeviceAuthenticationAttempts(username, address string, attempts int) error {
	return doSafeUpdate(context.Background(), deviceKey(username, address), false, func(gr *clientv3.GetResponse) (string, bool, error) {
		if len(gr.Kvs) != 1 {
			return "", false, errors.New("user device has multiple keys")
		}

		var device Device
		err := json.Unmarshal(gr.Kvs[0].Value, &device)
		if err != nil {
			return "", false, err
		}

		device.Attempts = attempts

		b, _ := json.Marshal(device)

		return string(b), false, err
	})
}

func GetAllDevices() (devices []Device, err error) {

	response, err := etcd.Get(context.Background(), "devices-", clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		return nil, err
	}

	for _, res := range response.Kvs {
		var device Device
		err := json.Unmarshal(res.Value, &device)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func AddDevice(username, address, publickey, preshared_key string) (Device, error) {
	if net.ParseIP(address) == nil {
		return Device{}, errors.New("Address '" + address + "' cannot be parsed as IP, invalid")
	}

	d := Device{
		Address:      address,
		Publickey:    publickey,
		Username:     username,
		PresharedKey: preshared_key,
	}

	b, _ := json.Marshal(d)
	key := deviceKey(username, address)

	_, err := etcd.Txn(context.Background()).Then(clientv3.OpPut(key, string(b)),
		clientv3.OpPut(fmt.Sprintf("deviceref-%s", address), key),
		clientv3.OpPut(fmt.Sprintf("deviceref-%s", publickey), key)).Commit()
	if err != nil {
		return Device{}, err
	}

	return d, err
}

func deviceKey(username, address string) string {
	return fmt.Sprintf("devices-%s-%s", username, address)
}

func DeleteDevice(username, id string) error {

	refKey := "deviceref-" + id

	realKey, err := etcd.Get(context.Background(), refKey)
	if err != nil {
		return err
	}

	if realKey.Count == 0 {
		return errors.New("no reference found")
	}

	deviceEntry, err := etcd.Get(context.Background(), string(realKey.Kvs[0].Value))
	if err != nil {
		return err
	}

	var d Device
	err = json.Unmarshal(deviceEntry.Kvs[0].Value, &d)
	if err != nil {
		return err
	}

	otherReferenceKey := "deviceref-" + d.Publickey
	if d.Publickey == id {
		otherReferenceKey = "deviceref-" + d.Address
	}

	_, err = etcd.Txn(context.Background()).Then(clientv3.OpDelete(string(realKey.Kvs[0].Value)), clientv3.OpDelete(refKey), clientv3.OpDelete(otherReferenceKey)).Commit()
	if err != nil {
		return err
	}

	return err
}

func DeleteDevices(username string) error {
	deleted, err := etcd.Delete(context.Background(), fmt.Sprintf("device-%s-", username), clientv3.WithPrefix())
	if err != nil {
		return err
	}

	ops := []clientv3.Op{}
	for _, reference := range deleted.PrevKvs {

		var d Device
		err := json.Unmarshal(reference.Value, &d)
		if err != nil {
			return err
		}

		ops = append(ops, clientv3.OpDelete("devicesref-"+d.Publickey), clientv3.OpDelete("deviceref-"+d.Address))
	}

	_, err = etcd.Txn(context.Background()).Then(ops...).Commit()
	return err
}

func UpdateDevicePublicKey(username, address string, publicKey wgtypes.Key) error {

	beforeUpadte, err := GetDeviceByAddress(address)
	if err != nil {
		return err
	}

	err = doSafeUpdate(context.Background(), deviceKey(username, address), false, func(gr *clientv3.GetResponse) (string, bool, error) {
		if len(gr.Kvs) != 1 {
			return "", false, errors.New("user device has multiple keys")
		}

		var device Device
		err := json.Unmarshal(gr.Kvs[0].Value, &device)
		if err != nil {
			return "", false, err
		}

		device.Publickey = publicKey.String()

		b, _ := json.Marshal(device)

		return string(b), false, err
	})

	if err != nil {
		return err
	}

	_, err = etcd.Delete(context.Background(), "devicesref-"+beforeUpadte.Publickey)

	return err
}

func GetDeviceByAddress(address string) (device Device, err error) {

	realKey, err := etcd.Get(context.Background(), "deviceref-"+address)
	if err != nil {
		return Device{}, err
	}

	if len(realKey.Kvs) != 1 {
		return Device{}, errors.New("incorrect number of keys for device reference")
	}

	response, err := etcd.Get(context.Background(), string(realKey.Kvs[0].Value))
	if err != nil {
		return Device{}, err
	}

	if len(response.Kvs) != 1 {
		return Device{}, errors.New("user device has multiple keys")
	}

	err = json.Unmarshal(response.Kvs[0].Value, &device)

	return
}

func GetDevicesByUser(username string) (devices []Device, err error) {

	response, err := etcd.Get(context.Background(), fmt.Sprintf("devices-%s-", username), clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		return nil, err
	}

	for _, res := range response.Kvs {
		var device Device
		err := json.Unmarshal(res.Value, &device)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
	}

	return
}
