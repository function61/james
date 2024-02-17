package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/function61/certbus/pkg/certbus"
	"github.com/function61/certbus/pkg/certificatestore"
	"github.com/function61/deployer/pkg/dstate"
	"github.com/function61/edgerouter/pkg/erconfig"
	"github.com/function61/edgerouter/pkg/erdiscovery/ehdiscovery"
	"github.com/function61/eventhorizon/pkg/ehclient"
	"github.com/function61/eventhorizon/pkg/ehdebug"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/bidipipe"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/osutil"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/james/pkg/domainwhois"
	"github.com/function61/james/pkg/duration"
	"github.com/function61/james/pkg/portainerclient"
	"github.com/function61/james/pkg/wsconnadapter"
	"github.com/function61/lambda-alertmanager/pkg/amstate"
	"github.com/function61/ubackup/pkg/ubconfig"
	"github.com/function61/ubackup/pkg/ubstorage"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

type std struct {
	PageName    string
	AppName     string
	GravatarUrl string
	UserEmail   string
}

func standardBs(pageName string) std {
	return std{
		PageName:    pageName,
		AppName:     "James",
		GravatarUrl: "https://www.gravatar.com/avatar/de2b26859f0a906bf1aa8fb0dc442842",
		UserEmail:   "joonas@joonas.fi",
	}
}

func webUiHandler(ctx context.Context, logger *log.Logger) (http.Handler, error) {
	routes := mux.NewRouter()

	templates, templateErr := template.New("templatecollection").Funcs(template.FuncMap{
		"dateinpast": func(ts time.Time) bool {
			return ts.Before(time.Now())
		},
		"joinstringlist": func(items []string) string {
			return strings.Join(items, ", ")
		},
		"agotime": func(ts time.Time) string {
			return duration.Humanize(time.Since(ts))
		},
		"frontenddescription": func(app erconfig.Application) string {
			return strings.TrimPrefix(app.Frontends[0].Describe(), "hostname:")
		},
		"humanizebytes": func(n int64) string {
			return fmt.Sprintf("%.2f GB", float64(n)/1024.0/1024.0/1024.0)
		},
		"backenddescription": func(app erconfig.Application) string {
			return app.Backend.Describe()
		},
	}).ParseGlob("templates/*.html")
	if templateErr != nil {
		return nil, templateErr
	}

	publicFiles := http.FileServer(http.Dir("./public/"))

	routes.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", publicFiles))

	routes.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = templates.Lookup("starter.html").Execute(w, struct {
			Std   std
			Names []string
		}{
			Std:   standardBs("Frontpage"),
			Names: []string{"joonas"},
		})
	})

	FIXME := "am:v1"

	tenantCtxSnapshots, err := ehreader.TenantCtxWithSnapshotsFrom(ehreader.ConfigFromEnv, FIXME)
	if err != nil {
		return nil, err
	}

	tenantCtx, err := ehreader.TenantCtxFrom(ehreader.ConfigFromEnv)
	if err != nil {
		return nil, err
	}

	deployer, err := dstate.LoadUntilRealtime(ctx, tenantCtx, logger)
	if err != nil {
		return nil, err
	}

	certificates, err := certbus.ResolveRealtimeState(ctx, *tenantCtx, logger)
	if err != nil {
		return nil, err
	}

	alertmanager, err := amstate.LoadUntilRealtime(
		ctx,
		tenantCtxSnapshots,
		logger)
	if err != nil {
		return nil, err
	}

	edgerouter, err := ehdiscovery.New(*tenantCtx, logger)
	if err != nil {
		return nil, err
	}

	jctx, err := readJamesfile()
	if err != nil {
		return nil, err
	}

	portainer, err := makePortainerClient2(ctx, *jctx)
	if err != nil {
		return nil, err
	}

	errAsInternalServerError := func(w http.ResponseWriter, err error) {
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	routes.HandleFunc("/edgerouter", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			apps, err := edgerouter.ReadApplications(r.Context())
			if err != nil {
				return err
			}

			return templates.Lookup("edgerouter.html").Execute(w, struct {
				Std  std
				Apps []erconfig.Application
			}{
				Std:  standardBs("Edgerouter"),
				Apps: apps,
			})
		}())
	})

	routes.HandleFunc("/edgerouter/{appId}/conf_dump", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			apps, err := edgerouter.ReadApplications(r.Context())
			if err != nil {
				return err
			}

			appId := mux.Vars(r)["appId"]

			app := func() *erconfig.Application {
				for _, app := range apps {
					if app.Id == appId {
						return &app
					}
				}

				return nil
			}()
			if app == nil {
				return errors.New("not found")
			}

			return jsonfile.Marshal(w, app)
		}())
	})

	routes.HandleFunc("/ubackup", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			config, err := ubconfig.ReadFromEnvOrFile()
			if err != nil {
				return err
			}

			storage, err := ubstorage.StorageFromConfig(config.Storage, nil)
			if err != nil {
				return err
			}

			services, err := storage.ListServices(r.Context())
			if err != nil {
				return err
			}

			return templates.Lookup("ubackup_select.html").Execute(w, struct {
				Std      std
				Services []string
			}{
				Std:      standardBs("µbackup"),
				Services: services,
			})
		}())
	})

	routes.HandleFunc("/ubackup/{service}", func(w http.ResponseWriter, r *http.Request) {
		serviceId := mux.Vars(r)["service"]

		errAsInternalServerError(w, func() error {
			config, err := ubconfig.ReadFromEnvOrFile()
			if err != nil {
				return err
			}

			storage, err := ubstorage.StorageFromConfig(config.Storage, nil)
			if err != nil {
				return err
			}

			backups, err := storage.List(r.Context(), serviceId)
			if err != nil {
				return err
			}

			return templates.Lookup("ubackup_list.html").Execute(w, struct {
				Std     std
				Service string
				Backups []ubstorage.StoredBackup
			}{
				Std:     standardBs("µbackup"),
				Service: serviceId,
				Backups: backups,
			})
		}())
	})

	routes.HandleFunc("/eventhorizon", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			streamName := func() string {
				if stream := r.URL.Query().Get("stream"); stream != "" {
					return stream
				} else {
					return "/"
				}
			}()

			resp, err := tenantCtx.Client.Read(r.Context(), ehclient.Beginning(streamName))
			if err != nil {
				return err
			}

			debug := &bytes.Buffer{}
			if err := ehdebug.Debug(resp, debug); err != nil {
				return err
			}

			return templates.Lookup("eventhorizon.html").Execute(w, struct {
				Std    std
				Output string
			}{
				Std:    standardBs("EventHorizon"),
				Output: debug.String(),
			})
		}())
	})

	routes.HandleFunc("/certbus", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			// TODO: refresh for certbus

			return templates.Lookup("certbus.html").Execute(w, struct {
				Std   std
				Certs []certificatestore.ManagedCertificate
			}{
				Std:   standardBs("SSL certificates"),
				Certs: certificates.All(),
			})
		}())
	})

	routes.HandleFunc("/deployer/releases", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			if err := deployer.Reader.LoadUntilRealtime(r.Context()); err != nil {
				return err
			}

			return templates.Lookup("deployer_releases.html").Execute(w, struct {
				Std      std
				Releases []dstate.SoftwareRelease
			}{
				Std:      standardBs("Releases"),
				Releases: deployer.State.AllNewestFirst(),
			})
		}())
	})

	routes.HandleFunc("/deployer/targets", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			if err := deployer.Reader.LoadUntilRealtime(r.Context()); err != nil {
				return err
			}

			releasesNewestFirst := deployer.State.AllNewestFirst()
			findReleaseByRepo := func(repo string) *dstate.SoftwareRelease {
				for _, release := range releasesNewestFirst {
					if release.Repository == repo {
						return &release
					}
				}

				return nil
			}

			type richTarget struct {
				Name            string
				DeployedVersion string
				LatestVersion   string
				AutoDeploy      bool
			}

			deployerWorkdir := "/vagrant/deployer/test1"
			deploymentsDir := filepath.Join(deployerWorkdir, "deployments/")

			targetDirs, err := ioutil.ReadDir(deploymentsDir)
			if err != nil {
				return err
			}

			// FIXME: this mapping should be automatic
			repoByTargetName := map[string]string{
				"varasto-updateserver": "function61/varasto",
				"varasto-docs":         "function61/varasto",
				"joonas.fi-blog":       "joonas-fi/joonas.fi-blog",
				"hq":                   "joonas-fi/hq",
			}

			richTargets := []richTarget{}

			for _, targetDir := range targetDirs {
				if !targetDir.IsDir() {
					continue
				}

				versionFile := struct {
					FriendlyVersion string `json:"friendly_version"`
				}{}

				versionJsonPath := filepath.Join(deploymentsDir, targetDir.Name(), "work", "version.json")

				if err := jsonfile.Read(versionJsonPath, &versionFile, false); err != nil {
					return err
				}

				repo, found := repoByTargetName[targetDir.Name()]
				if !found {
					return errors.New("repoByTargetName resolve failed")
				}

				latestVersion := ""

				latestRelease := findReleaseByRepo(repo)
				if latestRelease != nil {
					latestVersion = latestRelease.RevisionFriendly
				}

				richTargets = append(richTargets, richTarget{
					Name:            targetDir.Name(),
					DeployedVersion: versionFile.FriendlyVersion,
					LatestVersion:   latestVersion,
					AutoDeploy:      false,
				})
			}

			// /vagrant/deployer/test1/deployments

			return templates.Lookup("deployer_targets.html").Execute(w, struct {
				Std     std
				Targets []richTarget
			}{
				Std:     standardBs("Targets"),
				Targets: richTargets,
			})
		}())
	})

	routes.HandleFunc("/clusters", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			endpoints, err := portainer.ListEndpoints(r.Context())
			if err != nil {
				return err
			}

			return templates.Lookup("cluster_select.html").Execute(w, struct {
				Std       std
				Endpoints []portainerclient.Endpoint
			}{
				Std:       standardBs("Clusters"),
				Endpoints: endpoints,
			})
		}())
	})

	routes.HandleFunc("/clusters/{id}/stacks", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			endpointId, err := strconv.Atoi(mux.Vars(r)["id"])
			if err != nil {
				return err
			}

			stacks, err := portainer.ListStacks(r.Context())
			if err != nil {
				return err
			}

			filteredStacks := []portainerclient.Stack{}
			for _, stack := range stacks {
				if stack.EndpointID == endpointId {
					filteredStacks = append(filteredStacks, stack)
				}
			}

			return templates.Lookup("cluster_stacks.html").Execute(w, struct {
				Std    std
				Stacks []portainerclient.Stack
			}{
				Std:    standardBs("Stacks"),
				Stacks: filteredStacks,
			})
		}())
	})

	routes.HandleFunc("/clusters/{id}/stacks/{stackId}/composefile", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			stackFile, err := portainer.StackFile(r.Context(), mux.Vars(r)["stackId"])
			if err != nil {
				return err
			}

			w.Header().Set("Content-Type", "text/plain")
			_, err = fmt.Fprintf(w, "%s", stackFile)
			return err
		}())
	})

	routes.HandleFunc("/clusters/{id}/", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			endpointId, err := strconv.Atoi(mux.Vars(r)["id"])
			if err != nil {
				return err
			}

			stacks, err := portainer.ListStacks(r.Context())
			if err != nil {
				return err
			}

			filteredStacks := []portainerclient.Stack{}
			for _, stack := range stacks {
				if stack.EndpointID == endpointId {
					filteredStacks = append(filteredStacks, stack)
				}
			}

			return templates.Lookup("cluster_stacks.html").Execute(w, struct {
				Std    std
				Stacks []portainerclient.Stack
			}{
				Std:    standardBs("Stacks"),
				Stacks: filteredStacks,
			})
		}())
	})

	routes.HandleFunc("/dns", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			records, err := readDnsRecords()
			if err != nil {
				return err
			}

			return templates.Lookup("dns.html").Execute(w, struct {
				Std     std
				Records []dnsRecord
			}{
				Std:     standardBs("DNS records"),
				Records: records,
			})
		}())
	})

	routes.HandleFunc("/alertmanager/alerts", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			if err := alertmanager.Reader.LoadUntilRealtime(r.Context()); err != nil {
				return err
			}

			return templates.Lookup("alertmanager_alerts.html").Execute(w, struct {
				Std    std
				Alerts []amstate.Alert
			}{
				Std:    standardBs("Alerts"),
				Alerts: alertmanager.State.ActiveAlerts(),
			})
		}())
	})

	routes.HandleFunc("/alertmanager/httpmonitors", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			if err := alertmanager.Reader.LoadUntilRealtime(r.Context()); err != nil {
				return err
			}

			return templates.Lookup("alertmanager_httpmonitors.html").Execute(w, struct {
				Std      std
				Monitors []amstate.HttpMonitor
			}{
				Std:      standardBs("HTTP monitors"),
				Monitors: alertmanager.State.HttpMonitors(),
			})
		}())
	})

	routes.HandleFunc("/domains", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			return templates.Lookup("domains.html").Execute(w, struct {
				Std     std
				Domains []domainwhois.Data
			}{
				Std:     standardBs("Domains"),
				Domains: jctx.File.Domains,
			})
		}())
	})

	routes.HandleFunc("/alertmanager/deadmansswitches", func(w http.ResponseWriter, r *http.Request) {
		errAsInternalServerError(w, func() error {
			if err := alertmanager.Reader.LoadUntilRealtime(r.Context()); err != nil {
				return err
			}

			return templates.Lookup("alertmanager_deadmansswitches.html").Execute(w, struct {
				Std              std
				DeadMansSwitches []amstate.DeadMansSwitch
			}{
				Std:              standardBs("Dead man's switches"),
				DeadMansSwitches: alertmanager.State.DeadMansSwitches(),
			})
		}())
	})

	bash := exec.CommandContext(ctx, "bash")
	// we can't just connect web terminal's output to Bash's stdin, because Bash checks
	// isatty() and if it's not a "interactive terminal", it goes into automation-friendly
	// mode, disabling echos etc.
	bashStream, err := pty.Start(bash)
	if err != nil {
		return nil, err
	}

	/*
		bashStream := newProcessReadWriteAdapter(bash)
		if err := bash.Start(); err != nil {
			panic(err)
		}
	*/

	routes.HandleFunc("/term", func(w http.ResponseWriter, r *http.Request) {
		_ = templates.Lookup("term.html").Execute(w, struct {
			Std std
		}{
			Std: standardBs("Terminal"),
		})
	})

	routes.HandleFunc("/term/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}
		defer c.Close()

		if err := bidipipe.Pipe(bashStream, "bash", wsconnadapter.New(c), "websocket"); err != nil {
			log.Printf("ws ReadMessage: %v", err)
		}
	})

	return routes, nil
}

var upgrader = websocket.Upgrader{} // use default options

func webUi(ctx context.Context, logger *log.Logger) error {
	handler, err := webUiHandler(ctx, logger)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    ":80",
		Handler: handler,
	}

	tasks := taskrunner.New(ctx, logger)

	tasks.Start("listener "+srv.Addr, func(_ context.Context) error {
		return httputils.RemoveGracefulServerClosedError(srv.ListenAndServe())
	})

	tasks.Start("listenershutdowner", httputils.ServerShutdownTask(srv))

	return tasks.Wait()
}

type dnsRecord struct {
	Type    string
	Name    string
	Content string
}

func readDnsRecords() ([]dnsRecord, error) {
	tf := &TerraformFile{}
	if err := jsonfile.Read("../global/dns/terraform.tfstate", tf, false); err != nil {
		return nil, err
	}

	records := []dnsRecord{}

	for _, module := range tf.Modules {
		for _, resource := range module.Resources {
			if resource.Type != "cloudflare_record" {
				continue
			}

			records = append(records, dnsRecord{
				Type:    resource.Primary.Attributes["type"],
				Name:    resource.Primary.Attributes["hostname"],
				Content: resource.Primary.Attributes["value"],
			})
		}
	}

	return records, nil
}

func webUiEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "webui",
		Short: "Start web UI",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()

			osutil.ExitIfError(webUi(
				osutil.CancelOnInterruptOrTerminate(rootLogger),
				rootLogger))
		},
	}
}

type processReadWriteAdapter struct {
	// io.ReadCloser
	// io.WriteCloser
	stdout io.ReadCloser
	stdin  io.WriteCloser
}

/*
func newProcessReadWriteAdapter(cmd *exec.Cmd) io.ReadWriteCloser {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	return &processReadWriteAdapter{
		// ReadCloser:stdout,
		// WriteCloser:stdin,
		stdout: stdout,
		stdin:  stdin,
	}
}

func (p *processReadWriteAdapter) Read(buf []byte) (int, error) {
	return p.stdout.Read(buf)
}

func (p *processReadWriteAdapter) Write(buf []byte) (int, error) {
	if buf[0] == '\r' { // crap sandwich
		buf[0] = '\n'
	}

	return p.stdin.Write(buf)
}

func (f *processReadWriteAdapter) Close() error {
	errReaderClose := f.stdout.Close()
	if err := f.stdin.Close(); err != nil {
		return err
	}

	return errReaderClose
}
*/
