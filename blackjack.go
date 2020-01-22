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
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
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
	D           *sql.DB
	C           *spruce.Slot
	Se          *spruce.Hash
	randomMutex sync.Mutex
)

var (
	Rooms = make([]map[string]*websocket.Conn, 1024)

	Branker = make(map[string]string) // 这个是庄家的牌 每个庄家对应一个玩家
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

type UpS struct {
	Id   int
	Name string
	A    int
	S    int
	R    int
}
type Smsg struct {
	Id         int    `房间ID`
	Msg        string `消息提示`
	Type       int    `发牌1 2 结算3`
	Status     int    `房间状态`
	User       string `庄家标识`
	People     int    `当前房间多少人`
	Data       string `如果type是发牌 则是携带的 牌 ，反之则是结果`
	SelfStatus int    `自身状态` // -1 是等待 0 是死亡 1是继续
}
type BrankMsg struct {
	Id   int    `房间ID`
	Type int    `发牌1 2 结算3`
	User string `庄家标识`
	Data string `如果type是发牌 则是携带的 牌 ，反之则是结果`
}

func DoSend(ws *websocket.Conn, msg interface{}) {
	if err := websocket.JSON.Send(ws, msg); err != nil {
		log.Println(err)
	}
}
func webSocket(ws *websocket.Conn) {
	var err error
	ups := UpS{}
	for {
		if err = websocket.JSON.Receive(ws, &ups); err != nil {
			log.Println(err)
			break
		}
		defer ws.Close()
		fmt.Println("get msg:", ups)
		if Rooms[ups.Id] == nil {
			Rooms[ups.Id] = make(map[string]*websocket.Conn)
		}
		if k := Rooms[ups.Id][ups.Name]; k == nil {
			Rooms[ups.Id][ups.Name] = ws
		}
		for k, v := range Rooms[ups.Id] {
			msg := &Smsg{}
			msg.Id = ups.Id
			msg.People = len(Rooms[ups.Id])
			if ups.R == 1 { // 游戏中
				msg.Status = ups.R
				if ups.S == 1 {
					if k != ups.Name {
						msg.Msg = "玩家：" + ups.Name + " 出局"
					} else {
						msg.Msg = "你已经出局"
						fmt.Println(Branker[ups.Name])
						Branker[ups.Name] = ""
					}
					msg.Type = 3
					msg.Data = "you out"
					msg.SelfStatus = 0
					if v == nil {
						v.Close()
						delete(Rooms[ups.Id], k)
						return
					}
					go DoSend(v, msg)
				} else {
					if k == ups.Name {
						if ups.A == 1 { // 继续摸牌
							msg.Data = GetRandomInt(1, 10)
							msg.Type = 1
							msg.SelfStatus = 1
						} else if ups.A == 2 { // 首次拿牌
							oc := GetRandomInt(1, 10)
							ot := GetRandomInt(1, 10)
							Branker[ups.Name] = oc + "," + ot
							go DoSend(v, &BrankMsg{
								Id:   ups.Id,
								Type: 2,
								User: "brank_is_god",
								Data: `["` + oc + `","` + ot + `"]`,
							})
							msg.Data = `["` + GetRandomInt(1, 10) + `","` + GetRandomInt(1, 10) + `"]`
							msg.Type = 2
							msg.SelfStatus = 1
						} else if ups.A == 0 { //停止拿牌

						}
						if v == nil {
							v.Close()
							delete(Rooms[ups.Id], k)
							return
						}
						go DoSend(v, msg)
					}
				}
			} else { // 游戏还没开始 或者等待中 下一轮玩家可以等待至下一轮
				msg.Status = ups.R
				msg.Msg = "玩家：" + ups.Name + " 加入房间"
				msg.Type = 0
				msg.Data = "0"
				msg.SelfStatus = -1
				if v == nil {
					v.Close()
					delete(Rooms[ups.Id], k)
					return
				}
				go DoSend(v, msg)
			}
		}
	}
}
func GetRandomInt(start, end int) string {
	//访问加同步锁，是因为并发访问时容易因为时间种子相同而生成相同的随机数，那就狠不随机鸟！
	randomMutex.Lock()

	//利用定时器阻塞1纳秒，保证时间种子得以更改
	<-time.After(1 * time.Nanosecond)

	//根据时间纳秒（种子）生成随机数对象
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//得到[start,end]之间的随机数
	n := start + r.Intn(end-start+1)

	//释放同步锁，供其它协程调用
	randomMutex.Unlock()
	return strconv.Itoa(n)
}
