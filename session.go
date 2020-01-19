package blackjack

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"time"
)

func GetSession(key []byte) []byte {
	a := md5.New()
	a.Write(key)
	return Se.Get(a.Sum(nil))
}
func SetSession(key []byte, value []byte) {
	a := md5.New()
	a.Write(key)
	Se.Set(a.Sum(nil), value, 3600)
}
func IsZero(a interface{}) bool {
	t := reflect.TypeOf(a).Elem()
	v := reflect.ValueOf(a).Elem()
	for i := 0; i < t.NumField(); i++ {
		if v.Field(i).IsZero() {
			return false
		}
	}
	return true
}
func NewToken(s []byte) string {
	a := md5.New()
	a.Write(s)
	return fmt.Sprintf("%x", a.Sum([]byte(time.Now().String())))
}
