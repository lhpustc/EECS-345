package libkademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	mathrand "math/rand"
	"sss"
	"time"
	//"bytes"
)

type VanashingDataObject struct {
	AccessKey      int64
	Ciphertext     []byte
	NumberKeys     byte
	Threshold      byte
	timeoutSeconds int
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext
}

func (kk *Kademlia) VanishData(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	//get random key K
	cryptoKey := GenerateRandomCryptoKey()
	//Encrypting data
	cipherText := encrypt(cryptoKey, data)
	//splite K into N parts
	multiSssKeyMap, err := sss.Split(numberKeys, threshold, cryptoKey)
	if err != nil {
		return
	}
	//get access key L
	accessKey := GenerateRandomAccessKey()
	//get random location
	randIDs := CalculateSharedKeyLocations(accessKey, int64(numberKeys))
	//Store key
	i := 0
	for k, v := range multiSssKeyMap {

		all := append([]byte{k}, v...)
		kk.DoIterativeStore(randIDs[i], all)
		// fmt.Println("location  Vanish")
		// fmt.Println(randIDs[i])
		i++
	}

	vdo.AccessKey = accessKey
	vdo.Ciphertext = cipherText
	vdo.NumberKeys = numberKeys
	vdo.Threshold = threshold
	vdo.timeoutSeconds = timeoutSeconds
	fmt.Println("accessKey")
	fmt.Println(accessKey)
	return
}

func (kk *Kademlia) UnvanishData(vdo VanashingDataObject) (data []byte) {

	LocationIDs := CalculateSharedKeyLocations(vdo.AccessKey, int64(vdo.NumberKeys))
	multiSssKeyMap := make(map[byte][]byte)
	count := 0
	//get the map which contains (k, v)
	for _, id := range LocationIDs {
		all, err := kk.DoIterativeFindValue(id)
		if err == nil {
			multiSssKeyMap[all[0]] = all[1:]
			count++
		}
		if count == int(vdo.Threshold) {
			break
		}
	}
	fmt.Println(count)
	//check the piece we get is enough
	if count < int(vdo.Threshold) {
		return nil
	}
	//combine key piece and get cyptoKey
	cyptoKey := sss.Combine(multiSssKeyMap)
	//get data
	data = decrypt(cyptoKey, vdo.Ciphertext)
	return
}

func (kk *Kademlia) StoreVDO(vdoID ID, vdo VanashingDataObject) {
	sto := StoVDOType{vdoID, vdo}
	kk.StoVDOChan <- sto
}
