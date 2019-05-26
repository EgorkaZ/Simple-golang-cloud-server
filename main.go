package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"syscall"
	"time"
)

var currentDirectory = "./MyDirectory"

var commands = []string{"DirectoryName", "Delete", "NewName"}


type uploaded struct {
	name string
	body []byte
}

func find(b []byte, f []byte) int{
	l:= len(f)
	for i:=0;i<len(b);i++{
		if b[i] == f[0]{
			d := true
			for j:=1; j<l; j++{
				if b[i+j]!=f[j]{ d = false ; break}
			}
			if d {return i} else {d=true}
		}
	}
	return -1
}

func getCommand (b []byte) int{
	cLn := len(commands)
	for j:=0; j<cLn; j++{
		if b[0]==commands[j][0]{
			d:=byte(0)
			sLn:=len(commands[j])
			for k:=1+sLn%2; k<sLn; k=k+2{
				d += b[k]-commands[j][k]+b[k-1]-commands[j][k-1]
			}
			if d==0{
				return j
			}
		}
	}
	return -1
}

func parse(b []byte) uploaded {
	var file uploaded
	var i, r int
	l := len(b)
	for i=0; i<l&&b[i]!=13;i++{	}
	//if i==l{
	//	file.name = "#&login&#"
	//	file.body = b
	//	return file
	//}
	key:=b[0:i]
	end:=find(b[1:], key)

	i = find(b, []byte("filename"))+len("filename")
	for i=i+1; b[i]!='"'; i++{}
	for r=i+1; r<l&&b[r]!='"'; r++{}

	file.name = string(b[i+1:r])

	toNormal(&file.name)
	file.name = "/"+file.name

	for i=find(b, []byte("Content-Type")); b[i]!='\n'; i++{}
	i += 3
	file.body = b[i:end]
	if file.body[len(file.body)-1]==13{
		file.body = file.body[0:len(file.body)-1]
	}

	return file


}

func logIn(res http.ResponseWriter, req* http.Request){

	ck, err := req.Cookie("name")
	if !serveError(err){
		fmt.Println(ck.Value)
	}

	switch req.Method {
	case "GET":
		
		res.Header().Set("Content-type", "text/html")
		file, err := os.Open("./Pages/LogIn.txt")
		serveError(err)

		b, _ := ioutil.ReadAll(file)
		res.Write(b)
	case "POST":
		buf, err := ioutil.ReadAll(req.Body)
		serveError(err)
		hs := sha256.Sum256(buf)
		var cook http.Cookie
		cook.Name="name"
		cook.Value=hex.EncodeToString(hs[:])
		cook.Expires = time.Now().Add(30*time.Minute)
		cook.Path = "/"
		http.SetCookie(res, &cook)
		if cook.Value== "c2f4ed68dc50fb366ba6aee1366c838c0a637b2cfd512c2fa3c9a68fb7d6a974"{
			http.Redirect(res, req, "/home/", 302)
		} else {
			http.Redirect(res, req, "/login/", 303)
		}
	}
}

func makeDirectory(name string)  {
	os.Mkdir(currentDirectory+"/"+name, os.ModePerm)
	currentDirectory += "/"+name
}


func server (res http.ResponseWriter, req* http.Request){

	ck, err := req.Cookie("name")
	if err!=nil||ck.Value!="c2f4ed68dc50fb366ba6aee1366c838c0a637b2cfd512c2fa3c9a68fb7d6a974"{
		http.Redirect(res, req, "/login/", 301)
	}

	if req.Method=="POST"{
		buf, err:= ioutil.ReadAll(req.Body)
		serveError(err)

		switch getCommand(buf) { // get command in request body
		case -1: //None
			file := parse(buf)

			ofile, err := os.Create(currentDirectory + file.name)
			serveError(err)
			defer ofile.Close()

			_, err = ofile.Write(file.body)
			serveError(err)
		case 0: //DirectoryName
			makeDirectory(string(buf[len("DirectoryName="):]))
		case 1: //Delete
			var i int
			uri:=req.RequestURI
			for i=len(uri)-1; i>-1&&uri[i]!='/'; i--{}
			err := os.RemoveAll(currentDirectory+"/"+uri[i+1:])
			serveError(err)
			req.RequestURI = uri[:i+1]
		case 2: //NewName
			reqUrl, _ := url.ParseRequestURI(req.RequestURI)
			uri := reqUrl.Path

			ext := "."+getExtension(uri)
			var i int
			for i=len(uri)-1; i>-1&&uri[i]!='/'; i--{}
			i+=1;
			oldName := uri[i:]
			reqUrl, _ = url.ParseRequestURI("/"+string(buf[8:]))
			newName := reqUrl.Path[1:]

			os.Rename(currentDirectory+"/"+oldName, currentDirectory+"/"+newName+ext)
			req.RequestURI = uri[:i]
		}
	}


	//fileName := req.RequestURI[6:]

	reqUrl, _ := url.ParseRequestURI(req.RequestURI[5:])

	urlPath := reqUrl.Path

	stat, err := os.Stat(currentDirectory+ urlPath)
	if serveError(err){
		stat, err = os.Stat("./FileImage.png")
	}


	switch stat.IsDir()|| urlPath=="/parentDirectory" {
	case true:
		if urlPath!="/"{
			if urlPath =="/parentDirectory"{
				i := len(currentDirectory)
				for i=i-1; i>-1&&currentDirectory[i]!='/'; i--{}
				currentDirectory = currentDirectory[0:i]
				if currentDirectory=="."{
					currentDirectory="./MyDirectory"
				}

			}else {
				currentDirectory += urlPath
			}
		}
		fileStat, err := ioutil.ReadDir(currentDirectory)
		serveError(err)
		buf, err := ioutil.ReadFile("./Pages/Home.txt")
		serveError(err)

		page := string(buf)
		if currentDirectory!="./MyDirectory"{
			page += "<li><a href=\"/home/parentDirectory\">"+
				"<img src=\"https://www.pngrepo.com/png/213139/170/folder-ui.png\" alt = \"Folder image\"></a><p>..</p></li>"
		}

		for _, file := range fileStat{
			page+=linkMaker(file.Name())
		}
		page+="</ul></nav></body></html>"
		res.Header().Set("Content-type", "text/html")

		b := []byte(page)
		res.Write(b)
	case false:
		res.Header().Set("Content-Disposition", "attachment; filename = " + reqUrl.Path[1:])
		var file *os.File
		if reqUrl.Path == "/FileImage.png"{
			file, err = os.Open("./FileImage.png")
			serveError(err)
		}else{
			file, err = os.Open(currentDirectory + reqUrl.Path)
			serveError(err)
		}
		defer file.Close()
		b, _ := ioutil.ReadAll(file)
		res.Write(b)
		return
	}

}

func serveError(err error) bool{
	if err!=nil{
		fmt.Println(err)
	}
	return err!=nil
}

func linkMaker(fileName string) string {
	var s string
	ext:=getExtension(fileName)

	a, _ := syscall.UTF16FromString(fileName)
	fileName = ""
	if a[len(a)-1]==0 {a = a[0:len(a)-1]}
	for i:=0; i<len(a); i++{
		fileName += "&#" + strconv.Itoa(int(a[i])) +";"
	}

	if ext==""{
		s = "<li><a href=\"/home/"+fileName+"\">"+
			"<img src=\"https://www.pngrepo.com/png/213139/170/folder-ui.png\" alt = \"Folder image\"></a><p>" + fileName + "</p>" +
			"<ul class=\"sub\">"
	} else {
		if ext=="jpg"||ext=="png"{
			s = "<li><img src=\"" + fileName
		} else {
			s = "<li><img src=FileImage.png "
		}
		s += "\" alt=\"" +fileName + "\"><p>" +fileName+
			"</p><ul class = \"sub\">" +
			"<li><a href=\"/home/" + fileName+"\" >Download</a></li>"
	}

	s += "<li><form method=\"POST\" action = \"/home/"+fileName+ "\"><input type = \"text\" name = \"NewName\" style = \"width:100px;\"><input type = \"submit\" ></form></li>" +
		"<li><form method=\"POST\" action=\"/home/" + fileName + "\"><input type = \"submit\" value=\"Delete\" name=\"Delete\"></form></li>"+
		"</ul></li>"

	return s
}

func getExtension(file string) string{
	var i int
	for i=len(file)-1; i>-1&&file[i]!='.'; i--{}

	if i==-1{
		return ""
	}else {
		return file[i+1:]
	}
}

func toNormal(s *string)  {
	//l:= len(*s)
	a, _ := syscall.UTF16FromString(*s)

	*s = syscall.UTF16ToString(a)
}


func main() {

	http.HandleFunc("/home/", server)
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Println("root request")
		buf, _ :=  ioutil.ReadAll(req.Body)
		fmt.Println(string(buf))
		http.Redirect(res,req,"/home/", 301)
	})
	http.HandleFunc("/login/", logIn)
	http.HandleFunc("/favicon.ico", func(res http.ResponseWriter, requreq *http.Request) {
		fav, err := ioutil.ReadFile("./favicon.ico")
		if !serveError(err){
			res.Write(fav)
		}
	})

	err := http.ListenAndServe(":80", nil)
	if serveError(err){
		return
	}
}