package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"ops-api/config"
)

// Encrypt 字符串加密（使用默认公钥）
func Encrypt(str string) (string, error) {
	// 读取公钥文件
	publicKeyInterface := config.Conf.Settings["publicKey"]

	// 检查 publicKeyInterface 是否为 nil
	if publicKeyInterface == nil {
		return "", fmt.Errorf("publicKey is not set in configuration")
	}
	// 读取公钥文件
	publicKeySrt, ok := publicKeyInterface.(string)
	if !ok {
		return "", fmt.Errorf("expected string type for publicKey, got %T", publicKeyInterface)
	}

	// 公钥字符串转换为字节切片
	publicKeyData := []byte(publicKeySrt)

	// 解析公钥数据
	block, _ := pem.Decode(publicKeyData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return "", errors.New("invalid public key")
	}

	// 解析PEM格式的公钥
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 根据公钥加密
	encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), []byte(str))
	return base64.RawURLEncoding.EncodeToString(encryptedData), nil
}

// EncryptWithPublicKey 字符串加密（使用指定公钥）
func EncryptWithPublicKey(str, publicKey string) (string, error) {
	// 解析公钥数据
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		return "", errors.New("invalid public key")
	}

	// 解析PEM格式的公钥
	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 根据公钥加密
	encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, parsedKey.(*rsa.PublicKey), []byte(str))
	return base64.RawURLEncoding.EncodeToString(encryptedData), nil
}
