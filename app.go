package mobilicy

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	middleware []Handler
	routeStack map[Method][]*Route
	config     Config
	bot        *tgbotapi.BotAPI
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
	a.routeStack = make(map[Method][]*Route)
}

func (a *App) Command(path string, handlers ...Handler) Router {
	return a.Add(MethodCommand, path, handlers...)
}

func (a *App) Add(method Method, path string, handlers ...Handler) Router {
	return a.register(method, path, handlers...)
}

func (a *App) Use(handler Handler) {
	a.middleware = append(a.middleware, handler)
	for _, x := range a.routeStack {
		for _, y := range x {
			y.Handlers = append(y.Handlers, handler)
		}
	}
}

func (a *App) register(method Method, path string, handlers ...Handler) Router {
	r := Route{
		Method:   method,
		Path:     path,
		Handlers: append(a.middleware, handlers...),
	}
	a.addRoute(method, &r)
	return a
}

func (a *App) addRoute(m Method, r *Route) {
	a.routeStack[m] = append(a.routeStack[m], r)
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
		for _, route := range a.routeStack[m] {
			if route.match(u.Message.Command()) {
				if err := route.Handlers[0](ctx); err != nil {
					a.config.ErrHandler(ctx, err)
				}
				break
			}
		}
	}
}
