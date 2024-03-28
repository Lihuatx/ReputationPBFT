package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strconv"
)

// 如果当前目录下不存在目录Keys，则创建目录，并为各个节点生成rsa公私钥
func genRsaKeys(ClusterName string) {
	if !isExist("./Keys/" + ClusterName) {
		fmt.Println("检测到还未生成公私钥目录，正在生成公私钥 ...")
		err := os.Mkdir("Keys/"+ClusterName, 0644)
		if err != nil {
			log.Panic()
		}
		for i := 0; i <= 34; i++ {
			if !isExist("./Keys/" + ClusterName + "/" + ClusterName + strconv.Itoa(i)) {
				err := os.Mkdir("./Keys/"+ClusterName+"/"+ClusterName+strconv.Itoa(i), 0644)
				if err != nil {
					log.Panic()
				}
			}
			priv, pub := getKeyPair()
			privFileName := "./Keys/" + ClusterName + "/" + ClusterName + strconv.Itoa(i) + "/" + ClusterName + strconv.Itoa(i) + "_RSA_PIV"
			file, err := os.OpenFile(privFileName, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Panic(err)
			}
			defer file.Close()
			file.Write(priv)

			pubFileName := "./Keys/" + ClusterName + "/" + ClusterName + strconv.Itoa(i) + "/" + ClusterName + strconv.Itoa(i) + "_RSA_PUB"
			file2, err := os.OpenFile(pubFileName, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Panic(err)
			}
			defer file2.Close()
			file2.Write(pub)
		}
		fmt.Println("已为节点们生成RSA公私钥")
	}
}

// 生成rsa公私钥
func getKeyPair() (prvkey, pubkey []byte) {
	// 生成私钥文件
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey = pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	pubkey = pem.EncodeToMemory(block)
	return
}

// 判断文件或文件夹是否存在
func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println(err)
		return false
	}
	return true
}
