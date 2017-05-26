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

// TODO
// deixar getMany mais simples conforme
//   https://astaxie.gitbooks.io/build-web-application-with-golang/en/05.2.html
// usar decode uf8 sugerido pelo proprio goquery

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
const urlEvento string = "http://www.fecea.br/cursos/"

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
	// quero mecher nesse cara esta muito complexo
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

func remover(db *sql.DB, sqlRemove string, lst [][]string, lstIndices []int) {
	preRemove, err := db.Prepare(sqlRemove)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	defer preRemove.Close()

	for _, v := range lstIndices {

		valor, _ := strconv.Atoi(lst[v][0])
		_, err := preRemove.Exec(valor)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
	}

}

// inserirVazio sqlInsert deve ser parecido com "INSERT INTO `feed` VALUES (NULL, ?, ?, ?, ?)"
func inserirVazio(db *sql.DB, sqlInsert string, lstInterfaces [][]interface{}, lstIndices []int) {

	// eu acho q esse prepare eh uma vez so
	insertComm, err := db.Prepare(sqlInsert)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	defer insertComm.Close()

	for _, iv := range lstIndices {

		_, err = insertComm.Exec(lstInterfaces[iv]...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error()+"\n")
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

func extrairDatas(entrada string) (string, string) {

	ar := strings.Split(entrada, " ")
	primeiro := ar[0]
	ar2 := strings.Split(primeiro, "/")
	strtemplate := "%s-%s-%s"
	d1 := fmt.Sprintf(strtemplate, ar2[2], ar2[1], ar2[0])

	segundo := ar[2]
	ba := strings.Split(segundo, "/")
	d2 := fmt.Sprintf(strtemplate, ba[2], ba[1], ba[0])

	return d1, d2
}

// fin funcoes auxiliares
// ====================================

// feed faz: atualizar inserir no banco
func feed() {
	/*
		   casso ocora alguma falha nao pode parar tudo demo mostrar o log e a funcao deve retornar
		ordem feed:
		id, data_inclusao, texto, link_img, link
	*/
	confClear, err := bancoConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	db, err := sql.Open("mysql", confClear)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	defer db.Close()

	var lst [][]interface{}
	var ir = true
	resp, perr := http.Get(url)
	if perr != nil {
		fmt.Fprintln(os.Stderr, perr.Error())
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
		// id, data_inclusao, texto, link_img, link
		tmpit := make([]interface{}, 4)
		tmpit[0] = obj.data
		tmpit[1] = obj.texto
		tmpit[2] = obj.linkImg
		tmpit[3] = obj.link

		lst = append(lst, tmpit)

	})

	// apenas insere ou exclui nao atualiza
	if ir {

		fmt.Println("--feed")
		var lstRemover []int
		var lstAdicionar []int

		lstNomesPagina := make([]string, len(lst))
		i := 0
		for i < len(lst) {
			lstNomesPagina[i] = lst[i][1].(string)
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
		for i, valor := range lst {
			if !inArray(valor[1].(string), lstNomeTodosBanco) {
				lstAdicionar = append(lstAdicionar, i)
			}
		}

		// elementos que serao excluidos
		for i, valor := range lstTodos {
			if !inArray(valor[2], lstNomesPagina) {
				lstRemover = append(lstRemover, i)
			}
		}

		if len(lstAdicionar) > 0 {
			//convert Myfeed para []string
			fmt.Println("novos", len(lstAdicionar))
			sqlInserir := "INSERT INTO `feed` VALUES (NULL, ?, ?, ?, ?)"
			inserirVazio(db, sqlInserir, lst, lstAdicionar)
		}

		if len(lstRemover) > 0 {
			// codigo pra remover
			fmt.Println("remover", len(lstRemover))
			sqlRemover := "DELETE FROM `feed` WHERE id = ?"
			remover(db, sqlRemover, lstTodos, lstRemover)
		}

	} else {
		fmt.Println(url, " fora do ar")
	}
}

func evento() {
	/*
		   casso ocora alguma falha nao pode parar tudo demo mostrar o log e a funcao deve retornar

			 dados banco:
			 id, data_inclusao, nome, inicio_inscricao, fim_inscricao, link, vagas

			 curso que tiver vaga 0 sera excluido
	*/
	confClear, err := bancoConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	db, err := sql.Open("mysql", confClear)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	defer db.Close()

	var ir = true
	resp, perr := http.Get(urlEvento)
	if perr != nil {
		fmt.Fprintln(os.Stderr, perr.Error())
		ir = false
	}
	defer resp.Body.Close()

	stringPage := charmap.ISO8859_1.NewDecoder().Reader(resp.Body)
	stringPage2, _ := ioutil.ReadAll(stringPage)

	strReader := strings.NewReader(string(stringPage2))

	doc, err := goquery.NewDocumentFromReader(strReader)
	checkError(err)
	agora := time.Now().Format("2006-01-02 15:04:05")
	var dadosPagina [][]interface{}
	var inscricaoFull, vagas, nome, link string
	seletor := "#conteudo > table:nth-child(1) > tbody:nth-child(1) > tr:nth-child(2) > td:nth-child(1) > table:nth-child(1) > tbody:nth-child(1) > tr:nth-child(1) > td:nth-child(1) > table:nth-child(5) > tbody:nth-child(1) > tr:nth-child(4) > td:nth-child(1) > table:nth-child(1) > tbody:nth-child(1) tr"
	apartir := 2
	doc.Find(seletor).Each(func(i int, s *goquery.Selection) {
		/*

					indice 0
			<font size="2" face="Arial, Helvetica, sans-serif"><a href="index.php?p=5&amp;curso=194">Laboratório de Línguas: Facilitando e Desenvolvendo a Aprendizagem e Fluência da Língua Inglesa através do ensino da Gramática Básica</a></font>
			============
			indice 1

			============
			indice 2
			<font face="Arial, Helvetica, sans-serif">18/05/2017 até 01/06/2017</font>
			============
			indice 3
			<font face="Arial, Helvetica, sans-serif" color="#990000">Vagas Esgotadas</font>
			============
			indice 4
			<font face="Arial, Helvetica, sans-serif">Isenta</font>

		*/
		if i >= apartir {
			s.Find("td").Each(func(j int, s2 *goquery.Selection) {
				/*
					0 -> link com nome do curso e href com link do curso
					1 -> vazio
					2 -> periodo de inscreicao
					3 -> numero vargas
				*/

				if j == 0 {
					link, _ = s2.Find("a").Attr("href")
					nome = s2.Find("a").Text()
				} else if j == 2 {
					inscricaoFull = s2.Text()
				} else if j == 3 {
					vagas = s2.Text()
				}

			})
			// data_inclusao, nome, inicio_inscricao, fim_inscricao, link, vagas
			tmpOjb := make([]interface{}, 6) // campos do banco de dados menos o id
			d1, d2 := extrairDatas(inscricaoFull)
			nvagas, er := strconv.Atoi(vagas)
			if er != nil {
				nvagas = 0
			}
			tmpOjb[0] = agora
			tmpOjb[1] = nome
			tmpOjb[2] = d1
			tmpOjb[3] = d2
			tmpOjb[4] = link
			tmpOjb[5] = nvagas
			// fmt.Println("++++++++++++++")
			// fmt.Println("data =", agora, " nome =", nome, " link =", link, " inscricao =", inscricaoFull, " vagas =", vagas)
			// fmt.Println("**************")
			dadosPagina = append(dadosPagina, tmpOjb)

		}

	})

	// atualiza, exclui, insere
	if ir {

		fmt.Println("--evento")
		// esse esquema on array de int usar no feed
		// eh concerteza mais eficiente
		var lstAdicionar []int
		var lstAtualizar []int
		var lstRemover []int
		var lstExcluirNome []string

		lstNomesPagina := make([]string, len(dadosPagina))
		i := 0
		for i < len(dadosPagina) {
			lstNomesPagina[i] = dadosPagina[i][1].(string)
			i++
		}
		// id, data_inclusao, nome, inicio_inscricao, fim_inscricao, link, vagas

		sqlTodos := "SELECT * FROM `evento`"
		lstTodos := getMany(db, sqlTodos)
		var lstNomeTodosBanco []string
		for _, valor := range lstTodos {
			// contem apenas coluna texto
			lstNomeTodosBanco = append(lstNomeTodosBanco, valor[2])
		}

		nomeConectID := make(map[string]int)
		nomeConectIndice := make(map[string]int)
		for i, valor := range lstTodos {
			vi, _ := strconv.Atoi(valor[0])
			nomeConectID[valor[2]] = vi
			nomeConectIndice[valor[2]] = i
		}

		// novos, update
		for i, valor := range lstNomesPagina {
			qnt := dadosPagina[i][5].(int)
			if inArray(valor, lstNomeTodosBanco) {

				if qnt > 0 {
					lstAtualizar = append(lstAtualizar, i)
				} else {
					lstExcluirNome = append(lstExcluirNome, valor)
				}

			} else {
				if qnt > 0 {
					lstAdicionar = append(lstAdicionar, i)
				}
			}
		}

		// remover
		for i, valor := range lstTodos {
			if !inArray(valor[2], lstNomesPagina) {
				lstRemover = append(lstRemover, i)
			}
		}

		if len(lstRemover) > 0 || len(lstExcluirNome) > 0 {
			var quantideExcluir = len(lstRemover)
			if len(lstExcluirNome) > 0 {
				quantideExcluir += len(lstExcluirNome)
			}
			fmt.Println("remover", quantideExcluir)
			for _, v := range lstExcluirNome {
				lstRemover = append(lstRemover, nomeConectIndice[v])
			}
			sqlRemover := "DELETE FROM `evento` WHERE id = ?"
			remover(db, sqlRemover, lstTodos, lstRemover)
		}

		if len(lstAdicionar) > 0 {
			fmt.Println("novos", len(lstAdicionar))

			sqlInsert := "INSERT INTO `evento` VALUES (NULL, ?, ?, ?, ?, ?, ?)"
			inserirVazio(db, sqlInsert, dadosPagina, lstAdicionar)
		}

		if len(lstAtualizar) > 0 {
			fmt.Println("atualiza", len(lstAtualizar))

			sqlUpdate := "UPDATE `evento` SET `inicio_inscricao`=?" +
				", `fim_inscricao`=?, `link`=?, `vagas`=?" +
				" WHERE `id`=?"

			// tenho q alterar a lista de interface, deve ter (nessa ordem)
			// inicio_inscricao, fim_inscricao, link, vagas, id
			// **data_inclusao, nome, inicio_inscricao, fim_inscricao, link, vagas
			//     0             1         2               3            4      5
			itInsert := make([][]interface{}, len(dadosPagina))
			for j, valor := range dadosPagina {
				tmpi := make([]interface{}, 5)
				tmpi[0] = valor[2]                        // inicio_inscricao
				tmpi[1] = valor[3]                        // fim_inscricao
				tmpi[2] = valor[4]                        // link
				tmpi[3] = valor[5]                        // vagas
				tmpi[4] = nomeConectID[valor[1].(string)] // id(no banco de dados)

				itInsert[j] = tmpi
			}

			// geral o suficiente para se usado no update tmb
			inserirVazio(db, sqlUpdate, itInsert, lstAtualizar)
		}

	} else {
		fmt.Println(urlEvento, " fora do ar")
	}
}

func main() {

	evento()

	feed()

}
