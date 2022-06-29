package mobilicy

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hashicorp/go-multierror"
	"github.com/panjf2000/ants/v2"
	"log"
)

type Method int

const (
	MethodCommand Method = iota + 1
)

const defaultWorkPoolCap = 10

type Handler = func(*Ctx) error
type ErrHandler = func(*Ctx, error)

func defaultErrorHandler(ctx *Ctx, err error) {
	ctx.String(err.Error(), false)
}

type App struct {
	stack  map[Method][]*Route
	config Config
	bot    *tgbotapi.BotAPI
}

type Config struct {
	Token       string
	BotDebug    bool
	WorkPoolCap int
	ErrHandler  ErrHandler
}

func New(config Config) *App {
	app := &App{
		config: config,
	}
	if config.WorkPoolCap == 0 {
		app.config.WorkPoolCap = defaultWorkPoolCap
	}
	if config.ErrHandler == nil {
		app.config.ErrHandler = defaultErrorHandler
	}
	app.init()
	return app
}

func (a *App) init() {
	a.stack = make(map[Method][]*Route)
}

func (a *App) Command(path string, handlers ...Handler) Router {
	return a.Add(MethodCommand, path, handlers...)
}

func (a *App) Add(method Method, path string, handlers ...Handler) Router {
	return a.register(method, path, handlers...)
}

func (a *App) register(method Method, path string, handlers ...Handler) Router {
	r := Route{
		Method:   method,
		Path:     path,
		Handlers: handlers,
	}
	a.addRoute(method, &r)
	return a
}

func (a *App) addRoute(m Method, r *Route) {
	a.stack[m] = append(a.stack[m], r)
}

func (a *App) Run() error {
	bot, err := tgbotapi.NewBotAPI(a.config.Token)
	if err != nil {
		return err
	}
	bot.Debug = a.config.BotDebug
	a.bot = bot
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	return a.serve(updates)
}

func (a *App) serve(updates tgbotapi.UpdatesChannel) error {
	wp, err := ants.NewPoolWithFunc(a.config.WorkPoolCap, a.serveFunc)
	if err != nil {
		return err
	}
	defer wp.Release()
	for update := range updates {
		if err := wp.Invoke(update); err != nil {
			log.Println(err)
		}
	}
	wp.Waiting()
	return nil
}

func (a *App) serveFunc(i interface{}) {
	u, _ := i.(tgbotapi.Update)
	ctx := &Ctx{
		bot:    a.bot,
		update: u,
	}
	var m Method
	if u.Message.IsCommand() {
		m = MethodCommand
	}
	switch m {
	case MethodCommand:
		for _, route := range a.stack[m] {
			if route.match(u.Message.Command()) {
				var eg error
				for _, handler := range route.Handlers {
					if err := handler(ctx); err != nil {
						eg = multierror.Append(eg, err)
					}
				}
				if eg != nil {
					a.config.ErrHandler(ctx, eg)
				}
			}
		}
	}
}
