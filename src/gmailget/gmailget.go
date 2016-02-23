package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/mxk/go-imap/imap"
)

func check(s string, err error) {
	if err != nil {
		println(s)
		log.Fatal(err.Error())
	}
}

func main() {
	var (
		cmd *imap.Command
		rsp *imap.Response
	)
	imap_mailbox_arg := flag.String("imap_mailbox", "", "the IMAP mailbox to open")
	imap_user_arg := flag.String("imap_user", "", "the IMAP user")
	imap_password_arg := flag.String("imap_password", "", "the IMAP password")
	imap_server_arg := flag.String("imap_server", "", "the IMAP server")
	dest_dir_arg := flag.String("dest_dir", "", "the destination directory")

	flag.Parse()
	check_arg := func(a *string, s string) {
		if a == nil || *a == "" {
			fmt.Printf("Missing mandatory '-%s' parameter", s)
			os.Exit(1)
		}
	}
	check_arg(imap_mailbox_arg, "imap_mailbox")
	check_arg(imap_user_arg, "imap_user")
	check_arg(imap_password_arg, "imap_password")
	check_arg(imap_server_arg, "imap_server")
	imap_mailbox := *imap_mailbox_arg
	imap_user := *imap_user_arg
	imap_password := *imap_password_arg
	imap_server := *imap_server_arg
	dest_dir := *dest_dir_arg

	// Connect to the server
	println("Connecting to ", imap_server)
	c, err := imap.DialTLS(imap_server, nil)
	check("Connection failed", err)

	// Remember to log out and close the connection when finished
	defer c.Logout(30 * time.Second)

	// Print server greeting (first response in the unilateral server data queue)
	c.Data = nil

	if c.Caps["STARTTLS"] {
		c.StartTLS(nil)
	}

	if c.State() == imap.Login {
		c.Login(imap_user, imap_password)
	}

	c.Select(imap_mailbox, true)

	max_messages := uint32(100)
	set, _ := imap.NewSeqSet("")
	//nr_messages := max_messages
	if c.Mailbox.Messages >= max_messages {
		set.AddRange(1, 100)
	} else {
		//nr_messages = c.Mailbox.Messages
		set.Add("1:*")
	}
	cmd, err = c.Fetch(set, "BODY[]")
	check("FETCH error", err)

	emails := make([][]byte, 0)
	for cmd.InProgress() {
		// Wait for the next response (no timeout)
		c.Recv(-1)

		email_buf := bytes.Buffer{}
		// Process command data
		for _, rsp = range cmd.Data {
			email_buf.Write(imap.AsBytes(rsp.MessageInfo().Attrs["BODY[]"]))
		}
		if len(email_buf.Bytes()) > 0 {
			emails = append(emails, email_buf.Bytes())
		}
		cmd.Data = nil
		c.Data = nil
	}

	// Check command completion status
	if rsp, err := cmd.Result(imap.OK); err != nil {
		if err == imap.ErrAborted {
			fmt.Println("Fetch command aborted")
		} else {
			fmt.Println("Fetch error:", rsp.Info)
		}
	}
	bar := pb.StartNew(int(len(emails)))

	for i, e := range emails {
		f, err := os.Create(filepath.Join(dest_dir, fmt.Sprintf("%d", i)))
		check("Failed to create file", err)
		defer f.Close()

		bar.Increment()
		f.Write(e)
	}
	bar.FinishPrint("Done")
}
