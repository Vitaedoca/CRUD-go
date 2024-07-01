package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// Pessoa representa um indivíduo no sistema.
type Pessoa struct {
	ID   int    `json:"id"`
	Nome string `json:"nome"`
}

var dbConn *sql.DB

// configurarDB inicializa a conexão com o banco de dados e cria a tabela se não existir.
func configurarDB() {
	var err error
	connStr := "username:password@tcp(localhost:3306)/jean"
	dbConn, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Fatal("Erro ao abrir a conexão com o banco de dados:", err)
	}

	if err = dbConn.Ping(); err != nil {
		log.Fatal("Erro ao pingar o banco de dados:", err)
	}

	_, err = dbConn.Exec(`CREATE TABLE IF NOT EXISTS individuos (
		id INT AUTO_INCREMENT PRIMARY KEY,
		nome VARCHAR(255) NOT NULL
	)`)
	if err != nil {
		log.Fatal("Erro ao criar a tabela:", err)
	}
}

// listarPessoas responde com a lista de todas as pessoas.
func listarPessoas(w http.ResponseWriter, r *http.Request) {
	rows, err := dbConn.Query("SELECT id, nome FROM individuos")
	if err != nil {
		http.Error(w, "Erro ao buscar pessoas: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listaPessoas []Pessoa
	for rows.Next() {
		var p Pessoa
		if err := rows.Scan(&p.ID, &p.Nome); err != nil {
			http.Error(w, "Erro ao escanear pessoa: "+err.Error(), http.StatusInternalServerError)
			return
		}
		listaPessoas = append(listaPessoas, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listaPessoas)
}

// obterPessoa responde com os detalhes de uma pessoa pelo seu ID.
func obterPessoa(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	var p Pessoa
	err := dbConn.QueryRow("SELECT id, nome FROM individuos WHERE id = ?", id).Scan(&p.ID, &p.Nome)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Pessoa não encontrada", http.StatusNotFound)
		} else {
			http.Error(w, "Erro ao buscar pessoa: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// adicionarPessoa adiciona uma nova pessoa ao banco de dados.
func adicionarPessoa(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "O Content-Type deve ser application/json", http.StatusUnsupportedMediaType)
		return
	}

	var novaPessoa Pessoa
	err := json.NewDecoder(r.Body).Decode(&novaPessoa)
	if err != nil {
		http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	resultado, err := dbConn.Exec("INSERT INTO individuos (nome) VALUES (?)", novaPessoa.Nome)
	if err != nil {
		http.Error(w, "Erro ao inserir pessoa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := resultado.LastInsertId()
	if err != nil {
		http.Error(w, "Erro ao obter ID da nova pessoa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	novaPessoa.ID = int(id)
	json.NewEncoder(w).Encode(novaPessoa)
}

// removerPessoa deleta uma pessoa pelo seu ID.
func removerPessoa(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	_, err := dbConn.Exec("DELETE FROM individuos WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Erro ao deletar pessoa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// modificarPessoa atualiza o nome de uma pessoa pelo seu ID.
func modificarPessoa(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "O Content-Type deve ser application/json", http.StatusUnsupportedMediaType)
		return
	}

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var pessoaAtualizada Pessoa
	err = json.NewDecoder(r.Body).Decode(&pessoaAtualizada)
	if err != nil {
		http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	resultado, err := dbConn.Exec("UPDATE individuos SET nome = ? WHERE id = ?", pessoaAtualizada.Nome, id)
	if err != nil {
		http.Error(w, "Erro ao atualizar pessoa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	linhasAfetadas, err := resultado.RowsAffected()
	if err != nil {
		http.Error(w, "Erro ao verificar linhas afetadas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if linhasAfetadas == 0 {
		http.Error(w, "Pessoa não encontrada", http.StatusNotFound)
		return
	}

	pessoaAtualizada.ID = id
	json.NewEncoder(w).Encode(pessoaAtualizada)
}

// bemVindo responde com uma mensagem de boas-vindas.
func bemVindo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Bem-vindo ao nosso serviço!")
}

func main() {
	configurarDB()
	defer dbConn.Close()

	r := mux.NewRouter()

	r.HandleFunc("/", bemVindo)
	r.HandleFunc("/pessoas", listarPessoas).Methods(http.MethodGet)
	r.HandleFunc("/pessoas", adicionarPessoa).Methods(http.MethodPost)
	r.HandleFunc("/pessoas/{id}", obterPessoa).Methods(http.MethodGet)
	r.HandleFunc("/pessoas/{id}", removerPessoa).Methods(http.MethodDelete)
	r.HandleFunc("/pessoas/{id}", modificarPessoa).Methods(http.MethodPut)

	fmt.Println("Servidor em execução na porta 3333")
	err := http.ListenAndServe(":3333", r)
	if err != nil {
		fmt.Println("Erro ao iniciar o servidor:", err)
	}
}
