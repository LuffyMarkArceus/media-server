package storage

var DB = make(map[string]string)

func InitDB() {
	DB["shrey"] = "1234"
}
