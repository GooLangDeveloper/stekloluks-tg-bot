package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	botToken         string
	adminChatID      int64 = 433873179 // ID админа СТЕКЛОЛЮКС
	managerUsername  string = "@GXDVMN" // Ник менеджера

	pendingReminders = make(map[int64]time.Time)
	mu               sync.Mutex
)

func main() {
	// === ENV ===
	botToken = os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN не установлен. Укажите токен вашего бота.")
	}

	// === BOT INIT ===
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go reminderWorker(ctx, bot)

	log.Println("Bot Stekloluks started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Bot shutting down")
			return

		case update := <-updates:
			if update.Message != nil {
				handleMessage(bot, update.Message)
			}

			if update.CallbackQuery != nil {
				handleCallback(bot, update.CallbackQuery)
			}
		}
	}
}

// ================= HANDLERS =================

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// Обработка отправленного контакта
	if msg.Contact != nil {
		mu.Lock()
		delete(pendingReminders, chatID)
		mu.Unlock()

		// Пересылаем контакт админу
		forward := tgbotapi.NewForward(adminChatID, chatID, msg.MessageID)
		bot.Send(forward)

		// Подтверждение пользователю
		confirmText := fmt.Sprintf("Спасибо за обращение! Ваш контакт получен. Наш менеджер %s свяжется с вами в ближайшее время для обсуждения деталей поставки.", managerUsername)
		confirm := tgbotapi.NewMessage(chatID, confirmText)
		confirm.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		bot.Send(confirm)
		return
	}

	switch msg.Text {
	case "/start":
		sendStart(bot, chatID)
	case "/about":
		sendAbout(bot, chatID)
	case "/faq":
		sendFAQMenu(bot, chatID)
	case "/contact":
		requestContact(bot, chatID)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	if cb.Message == nil {
		return
	}

	chatID := cb.Message.Chat.ID

	switch cb.Data {
	case "start":
		sendStart(bot, chatID)

	case "about":
		sendAbout(bot, chatID)

	case "faq":
		sendFAQMenu(bot, chatID)

	case "contact":
		requestContact(bot, chatID)

	case "faq_1":
		sendText(bot, chatID, faq1())

	case "faq_2":
		sendText(bot, chatID, faq2())

	case "faq_3":
		sendText(bot, chatID, faq3())

	case "faq_4":
		sendText(bot, chatID, faq4())

	case "back":
		sendStart(bot, chatID)

	case "leave_contact":
		requestContact(bot, chatID)
	}

	// Отвечаем на коллбэк, чтобы убрать "часики" на кнопке в Telegram
	bot.Request(tgbotapi.NewCallback(cb.ID, ""))
}

// ================= UI =================

func sendStart(bot *tgbotapi.BotAPI, chatID int64) {
	text := `🏭 Добро пожаловать в СТЕКЛОЛЮКС!

Мы — ваш надежный промышленный партнёр в мире силиката. 

Специализируемся на производстве жидкого стекла и силикатной глыбы под любые задачи вашего производства с гарантией соблюдения ГОСТ и лабораторным контролем.

Выберите интересующий вас раздел ниже, чтобы узнать подробнее или запросить расчет стоимости.`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = mainMenu()
	bot.Send(msg)
}

func sendAbout(bot *tgbotapi.BotAPI, chatID int64) {
	text := `🏢 О компании СТЕКЛОЛЮКС

Мы обеспечиваем промышленные предприятия качественным силикатным сырьем в любых объемах. 

Наши главные принципы:
🔬 Собственный лабораторный контроль на каждом этапе.
📜 Строгое соответствие стандартам ГОСТ.
🏭 Готовность к промышленным объемам поставок.
🤝 Адаптация продукции конкретно под ваши производственные нужды.

Оставьте заявку, и мы подготовим для вас индивидуальное коммерческое предложение.`

	sendText(bot, chatID, text)
}

func sendFAQMenu(bot *tgbotapi.BotAPI, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Какую продукцию вы производите?", "faq_1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔬 Как проверяется качество?", "faq_2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚚 Какие объемы можете поставлять?", "faq_3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Как связаться напрямую?", "faq_4"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "back"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📋 Информация и частые вопросы:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func requestContact(bot *tgbotapi.BotAPI, chatID int64) {
	mu.Lock()
	pendingReminders[chatID] = time.Now()
	mu.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Нажмите кнопку ниже, чтобы поделиться контактом для связи с отделом продаж.")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📞 Отправить мой номер телефона"),
		),
	)
	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = true
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = backMenu()
	bot.Send(msg)
}

// ================= MENUS =================

func mainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Оставить заявку на расчет", "leave_contact"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏭 О компании", "about"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Продукция и FAQ", "faq"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("👨‍💼 Написать менеджеру", "https://t.me/GXDVMN"),
		),
	)
}

func backMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад в меню", "back"),
		),
	)
}

// ================= FAQ TEXTS =================

func faq1() string {
	return `📦 ПРОДУКЦИЯ

Мы производим и поставляем:
— Жидкое стекло
— Силикатную глыбу

Наше сырье отлично подходит под задачи производства строительных материалов, литейного производства, химической промышленности и других сфер.`
}

func faq2() string {
	return `🔬 КОНТРОЛЬ КАЧЕСТВА

Каждая партия сопровождается строгим лабораторным контролем. Мы гарантируем полное соответствие ГОСТ и предоставляем паспорта качества.

Физико-химические показатели силиката могут быть скорректированы индивидуально под ваши технологические процессы.`
}

func faq3() string {
	return `🚚 ОБЪЕМЫ ПОСТАВОК

Стекклюкс ориентирован на работу с B2B-сектором. Производственные мощности позволяют нам бесперебойно отгружать продукцию в промышленных объемах.

Мы ценим стабильность и готовы обсуждать долгосрочные контракты.`
}

func faq4() string {
	return fmt.Sprintf(`📞 СВЯЗЬ И ЗАКАЗЫ

Если у вас нестандартный запрос или вы хотите оперативно обсудить условия поставки, вы можете написать нашему менеджеру напрямую: %s

Или просто оставьте свой контакт через кнопку "Оставить заявку", и мы перезвоним в рабочее время.`, managerUsername)
}

// ================= REMINDER =================

func reminderWorker(ctx context.Context, bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			now := time.Now()

			mu.Lock()
			for chatID, ts := range pendingReminders {
				// Напоминание через 24 часа, если человек нажал "Оставить заявку", но не скинул контакт
				if now.Sub(ts) >= 24*time.Hour {
					text := fmt.Sprintf(`Здравствуйте! Вчера вы планировали оставить заявку на поставку силикатной продукции.

Если у вас остались вопросы по объемам, ГОСТам или ценам, наш менеджер готов проконсультировать вас напрямую: %s

Или нажмите кнопку ниже, чтобы мы сами связались с вами.`, managerUsername)

					msg := tgbotapi.NewMessage(chatID, text)
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("📝 Отправить контакт", "leave_contact"),
						),
					)

					bot.Send(msg)
					// Удаляем из мапы, чтобы не спамить каждый день
					delete(pendingReminders, chatID)
				}
			}
			mu.Unlock()
		}
	}
}
