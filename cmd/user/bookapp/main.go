package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"strings"

	// db "sigmaos/debug"
	"sigmaos/dbclnt"
	"sigmaos/dbd"
	"sigmaos/fslib"
	np "sigmaos/ninep"
	"sigmaos/proc"
	"sigmaos/procclnt"
)

//
// book web app, invoked by wwwd
//

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v args...\n", os.Args[0])
		os.Exit(1)
	}
	m, err := RunBookApp(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: error %v", os.Args[0], err)
		os.Exit(1)
	}
	s := m.Work()
	m.Exit(s)
}

type BookApp struct {
	*fslib.FsLib
	*procclnt.ProcClnt
	input  []string
	pipefd int
	dbc    *dbclnt.DbClnt
}

func RunBookApp(args []string) (*BookApp, error) {
	log.Printf("MakeBookApp: %v\n", args)
	ba := &BookApp{}
	ba.FsLib = fslib.MakeFsLib("bookapp")
	ba.ProcClnt = procclnt.MakeProcClnt(ba.FsLib)
	ba.input = strings.Split(args[1], "/")
	dbc, err := dbclnt.MkDbClnt(ba.FsLib, np.DBD)
	if err != nil {
		return nil, err
	}
	ba.dbc = dbc
	ba.Started()

	return ba, nil
}

func (ba *BookApp) writeResponse(data []byte) *proc.Status {
	_, err := ba.Write(ba.pipefd, data)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Pipe parse err %v\n", err), nil)
	}
	ba.Evict(proc.GetPid())
	return proc.MakeStatus(proc.StatusOK)
}

func (ba *BookApp) doView() *proc.Status {
	var books []dbd.Book
	err := ba.dbc.Query("select * from book;", &books)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Query err %v\n", err), nil)
	}
	t, err := template.New("test").Parse(`<h1>Books</h1><ul>{{range .}}<li><a href="http://localhost:8080/edit/{{.Title}}">{{.Title}}</a> by {{.Author}}</li> {{end}}</ul>`)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Template parse err %v\n", err), nil)
	}

	var data bytes.Buffer
	err = t.Execute(&data, books)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Template err %v\n", err), nil)
	}

	log.Printf("bookapp: html %v\n", string(data.Bytes()))
	return ba.writeResponse(data.Bytes())
}

func (ba *BookApp) doEdit(key string) *proc.Status {
	var books []dbd.Book
	q := fmt.Sprintf("select * from book where title=\"%v\";", key)
	err := ba.dbc.Query(q, &books)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Query err %v\n", err), nil)
	}
	t, err := template.New("edit").Parse(`<h1>Editing {{.Title}}</h1>
<form action="/save/{{.Title}}" method="POST">
<div><textarea name="title" rows="20" cols="80">{{printf "%s" .Title}}</textarea></div>
<div><input type="submit" value="Save"></div>
</form>`)
	var data bytes.Buffer
	err = t.Execute(&data, books[0])
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Template err %v\n", err), nil)
	}

	log.Printf("bookapp: html %v\n", string(data.Bytes()))
	return ba.writeResponse(data.Bytes())
}

func (ba *BookApp) doSave(key string, title string) *proc.Status {
	q := fmt.Sprintf("update book SET title=\"%v\" where title=\"%v\";", title, key)
	err := ba.dbc.Exec(q)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Query err %v\n", err), nil)
	}
	return proc.MakeStatusErr("Redirect", "/book/view/")
}

func (ba *BookApp) Work() *proc.Status {
	log.Printf("work %v\n", ba.input)
	fd, err := ba.Open(path.Join(proc.PARENTDIR, proc.SHARED)+"/", np.OWRITE)
	if err != nil {
		return proc.MakeStatusErr(fmt.Sprintf("Open err %v\n", err), nil)
	}
	ba.pipefd = fd
	defer ba.Close(fd)

	switch ba.input[0] {
	case "view":
		return ba.doView()
	case "edit":
		return ba.doEdit(ba.input[1])
	case "save":
		return ba.doSave(ba.input[1], os.Args[2])
	default:
		return proc.MakeStatusErr("File not found", nil)
	}
}

func (ba *BookApp) Exit(status *proc.Status) {
	log.Printf("bookapp exit %v\n", status)
	ba.Exited(status)
}
