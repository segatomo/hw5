package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/", handleExample)
	http.HandleFunc("/norikae", handleNorikae)
	// これをしないとcssなどのstaticファイルを読み込めない
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
}

// このディレクトリーに入っているすべての「.html」終わるファイルをtemplateとして読み込む。
var tmpl = template.Must(template.ParseGlob("*.html"))

// Templateに渡す内容を分かりやすくするためのtypeを定義しておきます。
// （「Page」という名前などは重要ではありません）。
type Page struct {
	A    string
	B    string
	Pata string
}

func handleExample(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// templateに埋める内容をrequestのFormValueから用意する。
	content := Page{
		A: r.FormValue("a"),
		B: r.FormValue("b"),
	}

	// とりあえずPataを簡単な操作で設定しますけど、すこし工夫をすれば
	// パタトクカシーーができます。

	var newPata string
	var aRune []rune
	var bRune []rune

	aRune = []rune(content.A)
	bRune = []rune(content.B)

	if len(aRune) > len(bRune) {
		for i := 0; i < len(bRune); i++ {
			newPata += string(aRune[i]) + string(bRune[i])
		}
		newPata += string(aRune[len(bRune):])
	} else {
		for i := 0; i < len(aRune); i++ {
			newPata += string(aRune[i]) + string(bRune[i])
		}
		newPata += string(bRune[len(aRune):])
	}

	content.Pata = newPata

	// example.htmlというtemplateをcontentの内容を使って、{{.A}}などのとこ
	// ろを実行して、内容を埋めて、wに書き込む。
	tmpl.ExecuteTemplate(w, "pata.html", content)
}

// LineはJSONに入ってくる線路の情報をtypeとして定義している。このJSON
// にこの名前にこういうtypeのデータが入ってくるということを表している。
type Line struct {
	Name     string
	Stations []string
}

// TransitNetworkは http://fantasy-transit.appspot.com/net?format=json
// の一番外側のリストのことを表しています。
type TransitNetwork struct {
	Network []Line
	From string
	To string
	AdjList map[string][]string
	Route []string
}

func handleNorikae(w http.ResponseWriter, r *http.Request) {
	// Appengineの「Context」を通してAppengineのAPIを利用する。
	ctx := appengine.NewContext(r)

	// clientはAppengine用のHTTPクライエントで、他のウェブページを読み込
	// むことができる。
	client := urlfetch.Client(ctx)

	// JSONとしての路線グラフ内容を読み込む
	resp, err := client.Get("http://fantasy-transit.appspot.com/net?format=json")
	if err != nil {
		panic(err)
	}

	// 読み込んだJSONをパースするJSONのDecoderを作る。
	decoder := json.NewDecoder(resp.Body)

	// JSONをパースして、「network」に保存する。
	var network TransitNetwork
	if err := decoder.Decode(&network.Network); err != nil {
		panic(err)
	}

	network.From = r.FormValue("fromsta")
	network.To = r.FormValue("tosta")

	adjList := makeAdj(r)
	network.AdjList = adjList

	route := bfs(network.AdjList, network.From, network.To)
	network.Route = route

	// handleExampleと同じようにtemplateにテンプレートを埋めて、出力する。
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "norikae.html", network)
}

// map[string][]string  
// {"渋谷": ["恵比寿", "原宿", "代官山", …], "目黒": ["五反田", "恵比寿", "不動前"…], …}
func bfs(adjList map[string][]string, start string, end string) []string {

    queue := make([][]string, 0)
    first := []string{start}
    queue = append(queue, first)

	visited := map[string]bool{}

    for !(len(queue) == 0) {
        route := queue[0]
        now := route[len(route) - 1]
        visited[now] = true

        if now == end {
            return route
        }

        _, exist := adjList[now]
        if exist {
            for i := 0; i < len(adjList[now]); i++ {
                next := adjList[now][i]
                if !visited[next] {
                    newRoute := append(route, next)
                    queue = append(queue, newRoute)
                }
            }
        }
        queue = queue[1:]
    }

	notFound := []string{"経路が見つかりませんでした"}
    return notFound
}

// Adj defines the contents of adjacency list
type Adj struct { 
	Start string
	Adj []string
}

// jsonデータを成形して  
// {"渋谷": ["恵比寿", "代々木", "代官山", …], "目黒": ["五反田", "恵比寿", "不動前"…], …}
// みたいな隣接リストを作る
func makeAdj(r *http.Request) map[string][]string {
	adjList := make(map[string][]string) 

	// Appengineの「Context」を通してAppengineのAPIを利用する。
	ctx := appengine.NewContext(r)

	// clientはAppengine用のHTTPクライエントで、他のウェブページを読み込
	// むことができる。
	client := urlfetch.Client(ctx)

	// JSONとしての路線グラフ内容を読み込む
	resp, err := client.Get("http://fantasy-transit.appspot.com/net?format=json")
	if err != nil {
		panic(err)
	}

	// 読み込んだJSONをパースするJSONのDecoderを作る。
	decoder := json.NewDecoder(resp.Body)

	// JSONをパースして、「network」に保存する。
	var network TransitNetwork
	if err := decoder.Decode(&network.Network); err != nil {
		panic(err)
	}

	for i := 0; i < len(network.Network); i++ {

		// name := network.Network[i].Name  // 路線名
		stations := network.Network[i].Stations  // 同じ路線の駅のリスト
		// if name == "山手線" {
		// 	stations = stations[1:]
		// }
		for idx, station := range stations {
			
			if _, ok := adjList[station]; ok {
				// すでに{"渋谷": ["恵比寿", "代々木"]}が存在していた時
				if idx > 0 {
					adjList[station] = append(adjList[station], stations[idx-1])
				}
				if idx < len(stations)-2 {
					adjList[station] = append(adjList[station], stations[idx+1])
				}
			} else {
				if idx > 0 && idx < len(stations)-2 {
					adjList[station] = []string{stations[idx-1], stations[idx+1]} 
				} else if idx > 0 {
					adjList[station] = []string{stations[idx-1]}
				} else if idx < len(stations)-2 { 
					adjList[station] = []string{stations[idx+1]}
				}
			}
		}
	}
	return adjList
}