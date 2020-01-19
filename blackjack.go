package blackjack

import (
	"./cedar"
	"./websockets"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tungyao/spruce"
	"github.com/tungyao/tjson"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func TEMPLATE(w http.ResponseWriter, path string) {
	fs, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return
	}
	o, _ := ioutil.ReadAll(fs)
	w.Write(o)
}
func READJSON(r *http.Request) (map[string]interface{}, error) {
	get := make([]byte, 1025)
	n, err := r.Body.Read(get)
	if err != io.EOF {
		log.Println(err)
		return nil, err
	}
	return tjson.Decode(get[:n])
}

var (
	D  *sql.DB
	C  *spruce.Slot
	Se *spruce.Hash
)

func init() {
	var err error
	D, err = sql.Open("mysql", fmt.Sprintf("%s:%s@%s(%s:%s)/%s?charset=utf8", "root", "1121331", "tcp", "localhost", "3306", "blog"))
	if err != nil {
		log.Println(err)
	}
	d := make([]spruce.DCSConfig, 1)
	d[0] = spruce.DCSConfig{
		Name:     "client0",
		Ip:       "127.0.0.1:90",
		Weigh:    0,
		Password: "",
	}
	C = spruce.StartSpruceDistributed(spruce.Config{
		ConfigType: spruce.MEMORY,
		Addr:       "127.0.0.1:90",
		DCSConfigs: d,
		KeepAlive:  true,
		IsBackup:   false,
		NowIP:      "127.0.0.1:90",
	})
	Se = spruce.CreateHash(1024)
}

type User struct {
	Id   int
	Name string
}

func Start() {
	r := cedar.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		TEMPLATE(w, "home.html")
	}, nil)
	r.Get("/login", func(writer http.ResponseWriter, request *http.Request) {
		TEMPLATE(writer, "./blackjack/login.html")
	}, nil)
	// 登录
	r.Post("/login_it", func(writer http.ResponseWriter, request *http.Request) {
		obj, err := READJSON(request)
		if err != nil {
			log.Println(err)
			return
		}
		sqls := "select id,name from user where name=" + obj["name"].(string) + " and pwd=" + obj["pwd"].(string)
		if s := C.Get(spruce.EntryHashGet([]byte(sqls))); len(s) != 0 {
			writer.Write(s)
			return
		}
		user := &User{}
		stmt, err := D.Prepare("select id,name from user where name=? and pwd=?")
		if err != nil {
			log.Println(err)
			return
		}
		row := stmt.QueryRow(obj["name"].(string), obj["pwd"].(string))
		row.Scan(&user.Id, &user.Name)
		if IsZero(user) {
			nt := NewToken([]byte(obj["name"].(string)))
			SetSession([]byte(nt), JsonEncode(user))
			writer.WriteHeader(200)
			writer.Write([]byte(nt))
			return
		}
		writer.WriteHeader(503)
	}, nil)
	r.Get("/register", func(writer http.ResponseWriter, request *http.Request) {
		TEMPLATE(writer, "./blackjack/register.html")
	}, nil)
	// 注册
	r.Post("/register_it", func(writer http.ResponseWriter, request *http.Request) {
		obj, err := READJSON(request)
		if err != nil {
			log.Println(err)
			return
		}
		stmt, err := D.Prepare("insert into user set name=?,email=?,pwd=?,status=1,type=1,create_time=?")
		if err != nil {
			log.Println(err)
			return
		}
		_, err = stmt.Exec(obj["name"], obj["email"], obj["pwd"], time.Now().Unix())
		if err != nil {
			log.Println(err)
			writer.WriteHeader(503)
			return
		}
		writer.Write([]byte("1"))
	}, nil)
	// 用来做长连接
	r.Get("/connect_room", nil, websockets.Handler(webSocket))
	r.Get("/static/", nil, http.StripPrefix("/static/", http.FileServer(http.Dir("./blackjack/static"))))
	if d := http.ListenAndServe(":80", r); d != nil {
		log.Println(d)
	}
}
func webSocket(ws *websockets.Conn) {
	var err error
	for {
		var reply string // get msg
		if err = websockets.Message.Receive(ws, &reply); err != nil {
			log.Println(err)
			continue
		}

		if err = websockets.Message.Send(ws, strings.ToUpper(reply)); err != nil {
			log.Println(err)
			continue
		}
	}
}
