package routing

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/client"
	"github.com/lloydmeta/tasques/client/tasks"
	"github.com/lloydmeta/tasques/example/ciphers/common"
	"github.com/lloydmeta/tasques/example/ciphers/server/config"
	"github.com/lloydmeta/tasques/example/ciphers/server/html"
	"github.com/lloydmeta/tasques/example/ciphers/server/persistence"
	"github.com/lloydmeta/tasques/models"
)

type Server struct {
	Config config.App
}

func (s *Server) Run() {
	esClient, err := s.Config.Elasticsearch.BuildClient()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	tasquesClient := s.Config.CipherQueuer.Server.BuildClient()

	messagesRepo := persistence.NewMessagesRepo(esClient)
	listTemplate, err := template.New("index.html").Parse(html.Index)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	errorTemplate, err := template.New("error.html").Parse(html.Error)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	controller := Controller{
		repo:          messagesRepo,
		queuerConfig:  s.Config.CipherQueuer,
		tasquesClient: tasquesClient,
		indexTemplate: listTemplate,
		errorTemplate: errorTemplate,
	}

	http.HandleFunc("/", controller.index)
	http.HandleFunc("/_create", controller.create)
	_ = http.ListenAndServe(s.Config.BindAddress, nil)
}

type Controller struct {
	repo          persistence.MessagesRepo
	queuerConfig  config.CipherQueuer
	tasquesClient *client.Tasques
	indexTemplate *template.Template
	errorTemplate *template.Template
}

func (c *Controller) index(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		messages, err := c.repo.List(r.Context())
		if err != nil {
			c.renderErr(w, err)
		} else {
			w.WriteHeader(http.StatusOK)
			view := IndexView{
				Created: nil,
				List:    messages,
			}
			_ = c.indexTemplate.Execute(w, view)
		}
	} else {
		http.NotFound(w, r)
	}
}

func (c *Controller) create(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			c.renderErr(w, err)
		} else {
			if messages, err := c.repo.List(r.Context()); err != nil {
				c.renderErr(w, err)
			} else {
				newMessage := common.NewMessage{Plain: r.PostForm.Get("Plain")}
				if inserted, err := c.repo.Create(r.Context(), newMessage); err != nil {
					c.renderErr(w, err)
				} else {
					if err := c.insertJobs(inserted); err != nil {
						c.renderErr(w, err)
					} else {
						messages := append([]common.Message{*inserted}, messages...)
						view := IndexView{
							Created: inserted,
							List:    messages,
						}
						_ = c.indexTemplate.Execute(w, view)
					}
				}
			}

		}
	} else {
		http.NotFound(w, r)
	}
}

func (c *Controller) renderErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_ = c.errorTemplate.Execute(w, err)
}

func (c *Controller) insertJobs(message *common.Message) error {
	var errs []error
	log.Info().Str("id", message.ID).Msg("Inserting Tasks")
	for _, cipher := range common.AllCiphers {
		log.Info().Str("cipher", cipher).Msg("Inserting Task")
		args := common.MessageArgs{
			MessageId: message.ID,
		}
		createTaskParams := tasks.NewCreateTaskParams()
		createTaskParams.NewTask = &models.TaskNewTask{
			Kind:  &cipher,
			Args:  args.ToMap(),
			Queue: &c.queuerConfig.Queue,
		}
		if _, err := c.tasquesClient.Tasks.CreateTask(createTaskParams); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		log.Error().
			Str("id", message.ID).
			Interface("errors", errs).
			Msg("Ran into issues with inserting Tasks")
		return fmt.Errorf("Lots of problems: [%v]", errs)
	} else {
		return nil
	}
}

type IndexView struct {
	Created *common.Message
	List    []common.Message
}
