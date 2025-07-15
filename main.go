package main

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"
)

var url = "https://raw.githubusercontent.com/GreatMedivack/files/master/list.out"

func main() {
	args := os.Args
	var server string
	if len(args) > 1 {
		server = args[1]
	} else {
		server = "test"
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()

	urlsplit := strings.Split(url, "/")
	filename := urlsplit[len(urlsplit)-1]

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	file.Close()

	list, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	date := time.Now().Format("01_02_2006")
	scanner := bufio.NewScanner(list)

	run, err := os.Create(fmt.Sprintf("%v_%v_running.out", server, date))
	if err != nil {
		log.Fatal(err)
	}
	defer run.Close()

	fail, err := os.Create(fmt.Sprintf("%v_%v_failed.out", server, date))
	if err != nil {
		log.Fatal(err)
	}
	defer fail.Close()

	runCount, failCount := 0, 0

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)

		sep := strings.Split(words[0], "-")
		name := words[0]
		if len(sep) > 2 && (len(sep[len(sep)-2]) == 9 || len(sep[len(sep)-2]) == 10) && (len(sep[len(sep)-1]) == 5) {
			name = strings.Join(sep[:len(sep)-2], "-")
		}

		switch words[2] {
		case "Running":
			io.WriteString(run, name+"\n")
			runCount++
		case "Error", "CrashLoopBackOff":
			io.WriteString(fail, name+"\n")
			failCount++
		}
	}

	report, err := os.Create(fmt.Sprintf("%v_%v_report.out", server, date))
	if err != nil {
		log.Fatal(err)
	}
	defer report.Close()
	os.Chmod(report.Name(), 0444)

	user, err := user.Current()
	rep_date := time.Now().Format("01/02/2006")
	text := fmt.Sprintf("Количество работающих сервисов: %d\nКоличество сервисов с ошибками: %d\nИмя системного пользователя: %s\nДата: %v",
		runCount, failCount, user.Username, rep_date)
	io.WriteString(report, text)

	os.Mkdir("archive", os.ModePerm)

	archPath := fmt.Sprintf("./archive/%s_%v.tar.gz", server, date)
	arch, err := os.Create(archPath)
	if err != nil {
		log.Fatal(err)
	}
	defer arch.Close()

	tw := tar.NewWriter(arch)
	defer tw.Close()

	err = archive(tw, run.Name())
	if err != nil {
		log.Fatal(err)
	}
	err = archive(tw, fail.Name())
	if err != nil {
		log.Fatal(err)
	}
	err = archive(tw, report.Name())
	if err != nil {
		log.Fatal(err)
	}

	os.Remove(run.Name())
	os.Remove(fail.Name())
	os.Remove(report.Name())
	os.Remove(filename)

	tr := tar.NewReader(arch)
	for {
		_, err = tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Задача успешно завершена")
}

func archive(tw *tar.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	header.Name = filename

	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}
	return nil
}
