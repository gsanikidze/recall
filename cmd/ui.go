package cmd

import (
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	"recall/internal/apiserver"
	uiassets "recall/ui"
)

// UI starts the web UI server on localhost:<port> (default 8888). It serves
// the REST API under /api/ and the React SPA under /.
func UI(args []string) error {
	fset := flag.NewFlagSet("ui", flag.ContinueOnError)
	port := fset.Int("port", 8888, "port to listen on")
	noBrowser := fset.Bool("no-browser", false, "do not open browser automatically")
	if err := fset.Parse(args); err != nil {
		return err
	}

	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	app := apiserver.New(e)
	app.Use("/", spaHandler())

	addr := fmt.Sprintf("localhost:%d", *port)
	url := "http://" + addr
	fmt.Fprintf(os.Stderr, "recall ui  →  %s\n", url)

	if !*noBrowser {
		go openBrowser(url)
	}

	return app.Listen(addr)
}

// spaHandler mounts the embedded React app with SPA fallback to index.html.
// If the UI has not been built (stub FS), returns 503.
func spaHandler() fiber.Handler {
	sub, err := fs.Sub(uiassets.FS, "dist")
	if err != nil {
		return func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusServiceUnavailable).
				SendString("recall UI not built.\nRun: make build")
		}
	}
	return filesystem.New(filesystem.Config{
		Root:         http.FS(sub),
		NotFoundFile: "index.html",
	})
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
