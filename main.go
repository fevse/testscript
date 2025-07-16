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
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	urlsplit := strings.Split(url, "/")
	filename := urlsplit[len(urlsplit)-1]

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Файл %v успешно скачан", url)

	list, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = list.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	date := time.Now().Format("02_01_2006")
	scanner := bufio.NewScanner(list)

	run, err := os.Create(fmt.Sprintf("%v_%v_running.out", server, date))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = run.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	fail, err := os.Create(fmt.Sprintf("%v_%v_failed.out", server, date))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = fail.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

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
			_, err = io.WriteString(run, name+"\n")
			if err != nil {
				log.Fatal(err)
			}
			runCount++
		case "Error", "CrashLoopBackOff":
			_, err = io.WriteString(fail, name+"\n")
			if err != nil {
				log.Fatal(err)
			}
			failCount++
		}
	}
	log.Printf("Файлы %v, %v успешно созданы", run.Name(), fail.Name())

	report, err := os.Create(fmt.Sprintf("%v_%v_report.out", server, date))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = report.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = os.Chmod(report.Name(), 0744)
	if err != nil {
		log.Fatal(err)
	}

	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	rep_date := time.Now().Format("02/01/2006")
	text := fmt.Sprintf("Количество работающих сервисов: %d\nКоличество сервисов с ошибками: %d\nИмя системного пользователя: %s\nДата: %v",
		runCount, failCount, user.Username, rep_date)
	_, err = io.WriteString(report, text)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Файл %v успешно создан", report.Name())

	err = os.Mkdir("archive", os.ModePerm)
	if err != nil {
		if err.Error() != "mkdir archive: file exists" {
			log.Fatal(err)
		}
	}

	archPath := fmt.Sprintf("./archive/%s_%v.tar.gz", server, date)
	arch, err := os.Create(archPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = arch.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	tw := tar.NewWriter(arch)
	defer func() {
		err = tw.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

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

	err = os.Remove(run.Name())
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(fail.Name())
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(report.Name())
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(filename)
	if err != nil {
		log.Fatal(err)
	}

	tr := tar.NewReader(arch)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Архив %v успешно создан", arch.Name())

	log.Println("Задача успешно завершена")
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
