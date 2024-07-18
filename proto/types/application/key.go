package application

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(appAddr string) []byte {
	var key []byte

	appAddrBz := []byte(appAddr)
	key = append(key, appAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
