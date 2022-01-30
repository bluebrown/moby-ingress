package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

func (hp *HaproxyConfig) Set(conf ConfigData) error {
	b := new(bytes.Buffer)
	err := hp.Template.Execute(b, conf)
	if err != nil {
		return err
	}
	hp.File = b.Bytes()
	hashBytes := md5.Sum(hp.File)
	hp.Hash = hex.EncodeToString(hashBytes[:])
	jsonBytes, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	hp.JSON = jsonBytes
	return nil
}
