package blackjack

import (
	"./cedar"
	"./spruce"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tungyao/tjson"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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
		TEMPLATE(w, "./blackjack/home.html")
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
		user := &User{}
		fmt.Println(obj)
		sqls := "select id,name from user where name=" + obj["name"].(string) + " and pwd=" + obj["pwd"].(string)
		if s := C.Get(spruce.EntryHashGet([]byte(sqls))); len(s) != 0 {
			u, err := tjson.Decode(s)
			if err != nil {
				log.Println(err)
				goto erox
			} else {
				idx, _ := strconv.Atoi(u["Id"].(string))
				user.Id = idx
				user.Name = u["Name"].(string)
			}
		} else {
			stmt, err := D.Prepare("select id,name from user where name=? and pwd=?")
			if err != nil {
				log.Println(err)
				return
			}
			defer stmt.Close()
			row := stmt.QueryRow(obj["name"].(string), obj["pwd"].(string))
			row.Scan(&user.Id, &user.Name)
			st, _ := json.Marshal(user)
			C.Set(spruce.EntryHashSet([]byte(sqls), st, 3600))

		}
	erox:
		stmt, err := D.Prepare("select id,name from user where name=? and pwd=?")
		if err != nil {
			log.Println(err)
			return
		}
		defer stmt.Close()
		row := stmt.QueryRow(obj["name"].(string), obj["pwd"].(string))
		row.Scan(&user.Id, &user.Name)
		st, _ := json.Marshal(user)
		C.Set(spruce.EntryHashSet([]byte(sqls), st, 3600))
		if IsZero(user) {
			nt := NewToken([]byte(obj["name"].(string)))
			SetSession([]byte(nt), JsonEncode(user))
			writer.WriteHeader(200)
			writer.Write([]byte(nt))
			return
		}
		writer.WriteHeader(503)
	}, nil)
	// 获取用户个人信息 用 token来换取 临时
	r.Post("/login_after", func(writer http.ResponseWriter, request *http.Request) {
		user := &User{}
		obj, err := READJSON(request)
		if err != nil {
			log.Println(err)
			writer.WriteHeader(503)
			return
		}
		x := GetSession([]byte(obj["token"].(string)))
		JsonDecode(user, x)
		d, err := json.Marshal(user)
		if err != nil {
			log.Println(err)
			writer.WriteHeader(503)
			return
		}
		writer.Write(d)
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
		stmt, err := D.Prepare("insert into user set name=?,email=?,pwd=?,status=1,type=1")
		if err != nil {
			log.Println(err)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(obj["name"], obj["email"], obj["pwd"])
		if err != nil {
			log.Println(err)
			writer.WriteHeader(503)
			writer.Write([]byte(err.Error()))
			return
		}
		writer.Write([]byte("1"))
	}, nil)
	// 创建房间
	r.Post("/create_room", func(writer http.ResponseWriter, request *http.Request) {
		obj, err := READJSON(request)
		if err != nil {
			log.Println(err)
			writer.WriteHeader(503)
			writer.Write([]byte(err.Error()))
			return
		}
		s := GetSession([]byte(obj["session_token"].(string)))
		x := &User{}
		JsonDecode(x, s)
		row := D.QueryRow("select id from twoone where create_user=?", x.Id)
		var id int64
		err = row.Scan(&id)
		if err != nil && id == 0 {
			res, err := D.Exec("insert into twoone set create_user=?,status=0,create_time=?", x.Id, time.Now().Unix())
			if err != nil {
				log.Println(err)
				writer.WriteHeader(503)
				writer.Write([]byte(err.Error()))
				return
			}
			id, _ = res.LastInsertId()
		}
		writer.Write([]byte(strconv.Itoa(int(id))))
	}, nil)
	// 用来做长连接
	r.Get("/connect_room", nil, websocket.Handler(webSocket))
	r.Get("/room", func(writer http.ResponseWriter, request *http.Request) {
		TEMPLATE(writer, "./blackjack/room.html")
	}, nil)
	r.Get("/static/", nil, http.StripPrefix("/static/", http.FileServer(http.Dir("./blackjack/static"))))
	if d := http.ListenAndServe(":80", r); d != nil {
		log.Println(d)
	}
}

var (
	Rooms = make([]map[string]*websocket.Conn, 1024)
)

type UpS struct {
	Id   int
	Name string
	A    int
	S    int
	r    int
}

func webSocket(ws *websocket.Conn) {
	var err error
	for {
		var reply string // get msg
		ups := UpS{}
		if err = websocket.Message.Receive(ws, &reply); err != io.EOF && err != nil {
			log.Println(err)
			break
		}
		fmt.Println(reply)
		err = json.Unmarshal([]byte(reply), &ups)
		if err != nil {
			log.Println("解析数据异常")
			break
		}
		if Rooms[ups.Id] == nil {
			Rooms[ups.Id] = make(map[string]*websocket.Conn)
		}
		if k := Rooms[ups.Id][ups.Name]; k == nil {
			Rooms[ups.Id][ups.Name] = ws
		}
		for _, v := range Rooms[ups.Id] {
			websocket.Message.Send(v, "asd")
		}
	}
}
