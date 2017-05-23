package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/text/encoding/charmap"
)

// Myjson corresponde ao tipo json do banco
type Myjson struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Database string `json:"database"`
}

// Myfeed representa um objeto feed para ser armazenado em banco
type Myfeed struct {
	data    string
	texto   string
	linkImg string
	link    string
}

// print objeto legivel stdout
func (f Myfeed) print() {
	template := "linkImg = %s, link = %s, texto = %s, data = %s"
	tbuild := fmt.Sprintf(template, f.linkImg, f.link, f.texto, f.data)
	fmt.Println(tbuild)
}

const url string = "http://www.fecea.br/"

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func feed() {
	/*
		  preciso: datade agora Y-m-d h:i:s
				link do post,
				link da img
				texto
	*/
	raw, err := ioutil.ReadFile("./config/banco.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var c Myjson
	json.Unmarshal(raw, &c)
	//fmt.Println(c.Database)

	confDb := "%s:%s@tcp(%s:3306)/%s"
	confClear := fmt.Sprintf(confDb, c.User, c.Password, c.Host, c.Database)
	// fmt.Println(confClear)

	db, err := sql.Open("mysql", confClear)
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()

	// fkAgora := "2017-04-10 20:38:03"
	// fkTexto := "blabla traquileba abalalibdudu :D"
	// fkLink := "fora do ar"
	// fkLinkImg := "img fora do ar"

	// quando for inserir tipo string nao precisa sercar de ''
	// exemplo de insert
	// insertComm, err := db.Prepare("INSERT INTO `feed` VALUES (NULL, ?, ?, ?, ?)")
	// if err != nil {
	// 	panic(err.Error())
	// }
	// defer insertComm.Close()
	//
	// _, err = insertComm.Exec(fkAgora, fkTexto, fkLinkImg, fkLink)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// select
	// rows, err := db.Query("SELECT * FROM feed")
	// if err != nil {
	// 	panic(err.Error()) // proper error handling instead of panic in your app
	// }
	//
	// // Get column names
	// columns, err := rows.Columns()
	// if err != nil {
	// 	panic(err.Error()) // proper error handling instead of panic in your app
	// }
	//
	// // Make a slice for the values
	// values := make([]sql.RawBytes, len(columns))
	//
	// // rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// // references into such a slice
	// // See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	// scanArgs := make([]interface{}, len(values))
	// for i := range values {
	// 	scanArgs[i] = &values[i]
	// }
	//
	// // Fetch rows
	// for rows.Next() {
	// 	// get RawBytes from data
	// 	err = rows.Scan(scanArgs...)
	// 	if err != nil {
	// 		panic(err.Error()) // proper error handling instead of panic in your app
	// 	}
	//
	// 	// Now do something with the data.
	// 	// Here we just print each column as a string.
	// 	var value string
	// 	for i, col := range values {
	// 		// Here we can check if the value is nil (NULL value)
	// 		if col == nil {
	// 			value = "NULL"
	// 		} else {
	// 			value = string(col)
	// 		}
	// 		fmt.Println(columns[i], ": ", value)
	// 	}
	// 	fmt.Println("-----------------------------------")
	// }
	// if err = rows.Err(); err != nil {
	// 	panic(err.Error()) // proper error handling instead of panic in your app
	// }

	resp, perr := http.Get(url)
	checkError(perr)
	defer resp.Body.Close()

	stringPage := charmap.ISO8859_1.NewDecoder().Reader(resp.Body)
	stringPage2, _ := ioutil.ReadAll(stringPage)

	strReader := strings.NewReader(string(stringPage2))

	doc, err := goquery.NewDocumentFromReader(strReader)
	checkError(err)

	seletor := "#main-area-1 > ul:nth-child(1) li"
	doc.Find(seletor).Each(func(i int, s *goquery.Selection) {
		var obj Myfeed
		agora := time.Now().Format("2006-01-02 15:04:05")
		obj.data = agora

		longo, ok := s.Find("a").Attr("href")
		if ok {
			obj.link = longo
		} else {
			obj.link = ""
		}

		alvo := s.Find("img")
		longo, ok = alvo.Attr("src")
		if ok {
			obj.linkImg = url + longo
		} else {
			obj.linkImg = ""
		}

		longo, ok = alvo.Attr("alt")
		if ok {
			obj.texto = longo
		} else {
			obj.texto = ""
		}

		obj.print()
		fmt.Println("--------------------------------------")

	})

}

func main() {

	feed()

	// o conteudo nao esta em utf8
	// acho q eh melchor comecar pelo arquivo
	// outro cara para fz parse de html https://godoc.org/go.marzhillstudios.com/pkg/go-html-transform/html/transform
	// resp, _ := http.Get("http://www.fecea.br/")
	// // bytes, _ := ioutil.ReadAll(resp.Body)
	// stringPage := charmap.ISO8859_1.NewDecoder().Reader(resp.Body)
	// stringPage2, _ := ioutil.ReadAll(stringPage)
	// fmt.Println("HTML:\n\n", string(stringPage2))
	//
	// resp.Body.Close()

	// --- Encoding: convert s from UTF-8 to ShiftJIS
	// declare a bytes.Buffer b and an encoder which will write into this buffer
	// var b bytes.Buffer
	// wInUTF8 := transform.NewWriter(&b, japanese.ShiftJIS.NewEncoder())
	// // encode our string
	// wInUTF8.Write([]byte(s))
	// wInUTF8.Close()
	// // print the encoded bytes
	// fmt.Printf("%#v\n", b)
	// encS := b.String()
	// fmt.Println(encS)
	//
	// // --- Decoding: convert encS from ShiftJIS to UTF8
	// // declare a decoder which reads from the string we have just encoded
	// rInUTF8 := transform.NewReader(strings.NewReader(encS), japanese.ShiftJIS.NewDecoder())
	// // decode our string
	// decBytes, _ := ioutil.ReadAll(rInUTF8)
	// decS := string(decBytes)
	// fmt.Println(decS)

}
