package blackjack

import (
	"fmt"
	"reflect"
	"strconv"
)

func JsonDecode(a interface{}, str []byte) {
	arg := SplitString(str, []byte{0})
	t := reflect.TypeOf(a).Elem()
	v := reflect.ValueOf(a).Elem()
	for i := 0; i < t.NumField(); i++ {
		switch v.Field(i).Kind() {
		case reflect.Int:
			it, _ := strconv.Atoi(string(arg[i]))
			v.Field(i).SetInt(int64(it))
		case reflect.String:
			v.Field(i).SetString(string(arg[i]))
		}
	}
}
func JsonEncode(a interface{}) []byte {
	d := make([][]byte, 0)
	v := reflect.ValueOf(a).Elem()
	t := reflect.TypeOf(a).Elem()
	for i := 0; i < t.NumField(); i++ {
		switch v.Field(i).Kind() {
		case reflect.Int:
			d = append(d, []byte(fmt.Sprintf("%d", v.Field(i).Int())))
		case reflect.String:
			d = append(d, []byte(fmt.Sprintf("%s", v.Field(i).String())))
		}
		d = append(d, []byte{0})
	}
	d = d[:len(d)-1]
	o := make([]byte, 0)
	for _, v := range d {
		for _, j := range v {
			o = append(o, j)
		}
	}
	return o
}
func SplitString(str []byte, p []byte) [][]byte {
	group := make([][]byte, 0)
	ps := 0
	for i := 0; i < len(str); i++ {
		if str[i] == p[0] && i < len(str)-len(p) {
			if len(p) == 1 {
				group = append(group, str[ps:i])
				ps = i + len(p)
				//return [][]byte{str[:i], str[i+1:]}
			} else {
				for j := 1; j < len(p); j++ {
					if str[i+j] != p[j] || j != len(p)-1 {
						continue
					} else {
						group = append(group, str[ps:i])
						ps = i + len(p)
					}
					//return [][]byte{str[:i], str[i+len(p):]}
				}
			}
		} else {
			continue
		}
	}
	group = append(group, str[ps:])
	return group
}
