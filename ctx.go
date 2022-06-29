package mobilicy

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Ctx struct {
	route   *Route
	bot     *tgbotapi.BotAPI
	update  tgbotapi.Update
	context context.Context
}

type Command struct {
	Command string
	Arg     string
}

func (c *Ctx) Update() tgbotapi.Update {
	return c.update
}

func (c *Ctx) FromUserID() int64 {
	return c.update.Message.From.ID
}

func (c *Ctx) FromChatID() int64 {
	return c.update.FromChat().ID
}

func (c *Ctx) FromMessageID() int {
	return c.update.Message.MessageID
}

func (c *Ctx) Context() context.Context {
	return c.context
}

func (c *Ctx) Command() *Command {
	if !c.update.Message.IsCommand() {
		return nil
	}
	return &Command{
		Command: c.update.Message.Command(),
		Arg:     c.update.Message.CommandArguments(),
	}
}

func (c *Ctx) String(s string, reply bool) {
	m := tgbotapi.NewMessage(c.FromChatID(), s)
	if reply {
		m.ReplyToMessageID = c.FromMessageID()
	}
	_, _ = c.bot.Send(m)
}
