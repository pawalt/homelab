package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
	"github.com/txn2/txeh"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	tex "text/template"
)

var (
	user = envOrDefault("GOLINKS_SSH_USERNAME", "root")
	remote = envOrDefault("GOLINKS_SSH_HOST", "openwrt")
	port = envOrDefault("GOLINKS_SSH_PORT", "22")
	hostsPath = envOrDefault("GOLINKS_HOSTPATH", "/etc/hosts.d/golinks.hosts")
)

func pullRedirects() (map[string]string, error) {
	config := &ssh.ClientConfig{
		User: user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// ssh conn
	conn, err := ssh.Dial("tcp", remote + ":" +port, config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// open source file
	srcFile, err := client.Open(hostsPath)
	if err != nil {
		return nil, err
	}

	// make tmp dest file
	dstFile, err := ioutil.TempFile("/tmp", "golinks")
	if err != nil {
		return nil, err
	}
	defer os.Remove(dstFile.Name())

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return nil, err
	}

	err = dstFile.Sync()
	if err != nil {
		return nil, err
	}

	hosts, err := txeh.NewHosts(&txeh.HostsConfig{ReadFilePath: dstFile.Name()})
	if err != nil {
		return nil, err
	}

	redirects := make(map[string]string)
	for _, line := range *hosts.GetHostFileLines() {
		comment := strings.TrimSpace(line.Comment)
		sides := strings.Split(comment, " -> ")
		if len(sides) != 2 {
			continue
		}

		source := sides[0]
		target := sides[1]
		redirects[source] = target

		if val, found := redirects[source]; found {
			log.Printf("attempted to map %v to %v but found %v\n", source, target, val)
			continue
		}

		redirects[source] = target
	}

	return redirects, nil
}

func renderHostsFile(newRedirects map[string]string) (string, error) {
	advertiseIP := GetOutboundIP()

	lines := make([]gin.H, 0, len(newRedirects))
	for source, dest := range newRedirects {
		host := strings.Split(source, "/")[0]
		comment := source + " -> " + dest

		lines = append(lines, gin.H{
			"ip": advertiseIP,
			"host": host,
			"comment": comment,
		})
	}

	lines = append(lines, gin.H{
		"ip": advertiseIP,
		"host": "go",
		"comment": "",
	})

	internaltempl := tex.Must(tex.New("").ParseFS(f, "templates/*.tmpl"))
	buf := new(bytes.Buffer)
	err := internaltempl.Lookup("hostsfile.tmpl").Execute(buf, gin.H{
		"lines": lines,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func writeHostsFile(hostfile string) error {
	config := &ssh.ClientConfig{
		User: user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// ssh conn
	conn, err := ssh.Dial("tcp", remote + ":" +port, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	defer client.Close()

	// open dest file
	dstFile, err := client.OpenFile(hostsPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE)
	if err != nil {
		return err
	}

	_, err = dstFile.Write([]byte(hostfile))
	if err != nil {
		return err
	}

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Run("kill -HUP $(ps | grep dnsmasq | grep -v grep | cut -d \" \" -f1)")

	return nil
}

func envOrDefault(envVar, def string) string {
	val, ok := os.LookupEnv(envVar)
	if ok {
		return val
	} else {
		return def
	}
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "100.100.100.100:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func writeRedirects(redirects map[string]string) error {
	hostfile, err := renderHostsFile(redirects)
	if err != nil {
		return fmt.Errorf("err rendering: %v", err)
	}

	err = writeHostsFile(hostfile)
	if err != nil {
		return fmt.Errorf("err writing: %v", err)
	}

	return nil
}
