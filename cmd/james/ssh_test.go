package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestDialSSHHostKeys(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	_, otherPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherSigner, err := ssh.NewSignerFromKey(otherPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name        string
		key         ssh.PublicKey
		host        string
		wantSuccess bool
	}{
		{name: "trusted", key: signer.PublicKey(), wantSuccess: true},
		{name: "changed", key: otherSigner.PublicKey()},
		{name: "unknown", key: signer.PublicKey(), host: "other.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("SSH_AUTH_SOCK", "")
			if err := os.Mkdir(filepath.Join(home, ".ssh"), 0700); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			host := test.host
			if host == "" {
				host = listener.Addr().String()
			}
			if err := os.WriteFile(filepath.Join(home, ".ssh", "known_hosts"), []byte(knownhosts.Line([]string{host}, test.key)+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			serverDone := make(chan struct{})
			go func() {
				defer close(serverDone)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				config := &ssh.ServerConfig{NoClientAuth: true}
				config.AddHostKey(signer)
				server, _, _, err := ssh.NewServerConn(conn, config)
				if err == nil {
					_ = server.Close()
				}
			}()
			client, err := dialSSH(listener.Addr().String(), "test")
			if (err == nil) != test.wantSuccess {
				t.Errorf("dialSSH error = %v, want success %v", err, test.wantSuccess)
			}
			if client != nil {
				_ = client.Close()
			}
			<-serverDone
		})
	}
}

func TestDialSSHMissingKnownHosts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := dialSSH("127.0.0.1:22", "test"); err == nil {
		t.Fatal("expected an error for missing known_hosts")
	}
}
