package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	tex "text/template"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	user      = envOrDefault("GOLINKS_SSH_USERNAME", "root")
	remote    = envOrDefault("GOLINKS_SSH_HOST", "openwrt")
	port      = envOrDefault("GOLINKS_SSH_PORT", "22")
	hostsPath = envOrDefault("GOLINKS_HOSTPATH", "/etc/hosts.d/golinks.hosts")
)

func renderHostsFile(newRedirects map[string]string) (string, error) {
	advertiseIP := getOutboundIP()

	lines := make([]gin.H, 0, len(newRedirects))
	for source, dest := range newRedirects {
		host := strings.Split(source, "/")[0]
		comment := source + " -> " + dest

		lines = append(lines, gin.H{
			"ip":      advertiseIP,
			"host":    host,
			"comment": comment,
		})
	}

	lines = append(lines, gin.H{
		"ip":      advertiseIP,
		"host":    "go",
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
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// ssh conn
	conn, err := ssh.Dial("tcp", remote+":"+port, config)
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

	session.Run("kill -HUP $(ps | grep dnsmasq | grep -v grep | xargs | cut -d \" \" -f1)")

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

func getOutboundIP() net.IP {
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

func refreshRedirects(conn *pgx.Conn) (map[string]string, error) {
	redirects, err := getRedirects(context.Background(), conn)
	if err != nil {
		return nil, err
	}

	err = writeRedirects(redirects)
	if err != nil {
		return nil, err
	}

	return redirects, nil
}

func getRedirects(ctx context.Context, conn *pgx.Conn) (map[string]string, error) {
	rows, err := conn.Query(ctx, "SELECT source, target FROM redirects")
	if err != nil {
		return nil, err
	}

	redirects := make(map[string]string)
	// there has to be a better way to do this right i feel crazy
	for rows.Next() {
		var source string
		var target string
		err := rows.Scan(&source, &target)
		if err != nil {
			return nil, err
		}
		redirects[source] = target
	}
	return redirects, nil
}
