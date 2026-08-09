package feishu

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// RegisterOptions controls the QR scan-to-create flow.
type RegisterOptions struct {
	Domain         string // feishu (default) or lark
	Timeout        time.Duration
	Stdout         io.Writer
	SkipQRRender   bool // if true, only print URL
	FromTag        string
}

// QRRegister runs init → begin → print QR → poll → optional bot probe.
// Aligns with Hermes plugins/platforms/feishu qr_register.
func QRRegister(ctx context.Context, opts RegisterOptions) (*Credentials, error) {
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	domain := opts.Domain
	if domain == "" {
		domain = "feishu"
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	fmt.Fprint(out, "Connecting to Feishu / Lark... ")
	if err := InitRegistration(ctx, domain); err != nil {
		fmt.Fprintln(out, "failed")
		return nil, err
	}
	begin, err := BeginRegistration(ctx, domain)
	if err != nil {
		fmt.Fprintln(out, "failed")
		return nil, err
	}
	fmt.Fprintln(out, "done.")
	fmt.Fprintln(out)

	if !opts.SkipQRRender {
		if err := RenderQR(out, begin.QRURL); err != nil {
			fmt.Fprintf(out, "  (QR render skipped: %v)\n", err)
		}
	}
	fmt.Fprintf(out, "\n  Scan the QR code with Feishu / Lark mobile app,\n  or open this URL:\n\n  %s\n\n", begin.QRURL)
	if begin.UserCode != "" {
		fmt.Fprintf(out, "  User code: %s\n\n", begin.UserCode)
	}
	fmt.Fprint(out, "  Waiting for scan")

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Progress dots in a side goroutine until poll returns.
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				fmt.Fprint(out, ".")
			case <-pollCtx.Done():
				return
			}
		}
	}()

	expire := begin.ExpireIn
	if d := int(timeout.Seconds()); d > 0 && d < expire {
		expire = d
	}
	creds, err := PollRegistration(pollCtx, begin.DeviceCode, begin.Interval, expire, domain)
	close(done)
	fmt.Fprintln(out)
	if err != nil {
		return nil, err
	}

	name, oid, perr := ProbeBot(ctx, creds.AppID, creds.AppSecret, creds.Domain)
	if perr == nil {
		creds.BotName = name
		creds.BotOpenID = oid
	}
	return creds, nil
}

// RenderQR writes a terminal ASCII QR for url.
func RenderQR(w io.Writer, url string) error {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return err
	}
	// ToSmallString(false) = black modules on white (invert for light terminals).
	_, err = io.WriteString(w, q.ToSmallString(false))
	return err
}
