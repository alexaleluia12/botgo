package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
)

// essa eh a primeira coisa q fz em Go
// codigo nao vai estar muito bom
// talvez no futuro eu mude as coisas
// so quero que isso funcione por enquanto, eu consiga fz deploy para server

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

const url string = "http://www.fecea.br/"

// ===============================
// inicio funcoes auxiliares

// print objeto legivel stdout
func (f Myfeed) print() {
	template := "linkImg = %s, link = %s, texto = %s, data = %s"
	tbuild := fmt.Sprintf(template, f.linkImg, f.link, f.texto, f.data)
	fmt.Println(tbuild)
}

// getMany retorna matriz com os valores na ordem de sqlQuery
func getMany(db *sql.DB, sqlQuery string) [][]string {

	var lstOut [][]string
	rows, err := db.Query(sqlQuery)

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		var tmpLst []string
		var value string
		for _, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			// fmt.Println(columns[i], ": ", value)
			tmpLst = append(tmpLst, value)
		}
		lstOut = append(lstOut, tmpLst)

	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return lstOut
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func bancoConfig() (string, error) {
	raw, err := ioutil.ReadFile("./config/banco.json")
	if err != nil {
		return "", err
	}

	var c Myjson
	json.Unmarshal(raw, &c)

	confDb := "%s:%s@tcp(%s:3306)/%s"
	confClear := fmt.Sprintf(confDb, c.User, c.Password, c.Host, c.Database)

	return confClear, nil
}

func inArray(alvo string, lst []string) bool {
	for _, valor := range lst {
		if valor == alvo {
			return true
		}
	}

	return false
}

func remover(db *sql.DB, sqlRemove string, lst [][]string) {
	preRemove, err := db.Prepare(sqlRemove)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return
	}
	defer preRemove.Close()

	for _, v := range lst {

		valor, _ := strconv.Atoi(v[0])
		_, err := preRemove.Exec(valor)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
	}

}

// inserirVazio sqlInsert deve ser parecido com "INSERT INTO `feed` VALUES (NULL, ?, ?, ?, ?)"
func inserirVazio(db *sql.DB, sqlInsert string, lst [][]string) {
	lstInterfaces := make([][]interface{}, len(lst))
	for iv, v := range lst {
		tmpi := make([]interface{}, len(v))
		for i, k := range v {
			tmpi[i] = k
		}
		lstInterfaces[iv] = tmpi
	}

	// eu acho q esse prepare eh uma vez so
	insertComm, err := db.Prepare(sqlInsert)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		return
	}
	defer insertComm.Close()

	for _, iv := range lstInterfaces {
		_, err = insertComm.Exec(iv...)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error()+"\n")
			return
		}
	}
}

// MyfeedToArrString convert
func MyfeedToArrString(m []Myfeed) [][]string {
	tmpadd := make([][]string, len(m))
	// elemento deve ser inseridos na msm ordem do insert
	for i, v := range m {
		tmparr := make([]string, 4) // quantide de atributos de Myfeed
		tmparr[0] = v.data
		tmparr[1] = v.texto
		tmparr[2] = v.linkImg
		tmparr[3] = v.link

		tmpadd[i] = tmparr
	}

	return tmpadd
}

// fin funcoes auxiliares
// ====================================

// feed faz: atualizar inserir no banco
func feed() {
	/*
	   casso ocora alguma falha nao pode parar tudo demo mostrar o log e a funcao deve retornar
	*/
	confClear, err := bancoConfig()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	db, err := sql.Open("mysql", confClear)
	if err != nil {
		fmt.Println(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		os.Exit(1)
	}
	defer db.Close()

	// fkAgora := "2017-04-10 20:38:03"
	// fkTexto := "aaaalexxxx eh um cusao"
	// fkLink := "fora do A R"
	// fkLinkImg := "img fora do ar :kkkkjjkkjj"
	//
	// akteste := [4]string{fkAgora, fkTexto, fkLink, fkLinkImg}
	// arin := make([]interface{}, len(akteste))
	// for i, v := range akteste {
	// 	arin[i] = v
	// }
	// // quando for inserir tipo string nao precisa sercar de ''
	// // exemplo de insert
	// insertComm, err := db.Prepare("INSERT INTO `feed` VALUES (NULL, ?, ?, ?, ?)")
	// if err != nil {
	// 	panic(err.Error())
	// }
	// defer insertComm.Close()
	//
	// _, err = insertComm.Exec(arin...)
	// if err != nil {
	// 	panic(err.Error())
	// }

	var lst []Myfeed
	var ir = true
	resp, perr := http.Get(url)
	if perr != nil {
		fmt.Println(perr.Error())
		ir = false
	}
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

		lst = append(lst, obj)

	})

	// apenas insere ou exclui nao atualiza
	if ir {

		fmt.Println("--feed")
		var lstRemover [][]string
		var lstAdicionar []Myfeed

		lstNomesPagina := make([]string, len(lst))
		i := 0
		for i < len(lst) {
			lstNomesPagina[i] = lst[i].texto
			i++
		}

		sqlTodos := "SELECT * FROM `feed` ORDER BY `texto`"
		lstTodos := getMany(db, sqlTodos)
		var lstNomeTodosBanco []string
		for _, valor := range lstTodos {
			// contem apenas coluna texto
			lstNomeTodosBanco = append(lstNomeTodosBanco, valor[2])
		}

		// novos elementos
		for _, valor := range lst {
			if !inArray(valor.texto, lstNomeTodosBanco) {
				lstAdicionar = append(lstAdicionar, valor)
			}
		}

		// elementos que serao excluidos
		// lstRemover contem elementos do banco pq preciso da id deles para remover
		for _, valor := range lstTodos {
			if !inArray(valor[2], lstNomesPagina) {
				lstRemover = append(lstRemover, valor)
			}
		}

		if len(lstAdicionar) > 0 {
			//convert Myfeed para []string
			fmt.Println("novos", len(lstAdicionar))
			tmpadd := MyfeedToArrString(lstAdicionar)
			sqlInserir := "INSERT INTO `feed` VALUES (NULL, ?, ?, ?, ?)"
			inserirVazio(db, sqlInserir, tmpadd)
		}

		if len(lstRemover) > 0 {
			// codigo pra remover
			fmt.Println("remover", len(lstRemover))
			sqlRemover := "DELETE FROM `feed` WHERE id = ?"
			remover(db, sqlRemover, lstRemover)
		}

		// tarefa
		// lista para esses dois selects
		// vou usar to sql_todos, na hora de excluir
		// verificar se ha ocrrencia de uma string num array
		// sql_todos = "SELECT * FROM `feed` ORDER BY `texto`"
		// nome_todos = "SELECT `texto` FROM `feed` ORDER BY `texto`"
		//
	}
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
