# gopray 🙏

A Telegram Bot to get muslim prayers time, And subscribe to get notified on the prayers time 

## Contributing 🤼

Before we start talking about the bot, I would really appreciate if you can contribute to this project by adding more features or fixing bugs. It is totally open source and free to use. 😁

## Usecase 🛠️

```mermaid
graph LR
A[User] --> |Get Prayers Time| B((Bot))
A[User] --> |Subscribe To Bot| B((Bot))
A[User] --> |Change Bot Language| B((Bot))
B((Bot)) --> |Notifiy Users about prayers| B((Bot))
A[User] --> |Send Feedback Messages\nOr Bug Reports| B((Bot))
B((Bot)) --> |Store Prayers Time| C[[Database]]
B((Bot)) --> |Store User ID For Subscribtion| C[[Database]]
```

## Subscribe Feature 📢

This feature allows the bot to notify users `20 min` before the prayers time.

Users can subscribe to the bot by sending `/subscribe` command to the bot. And unsubscribe by sending `/unsubscribe` command to the bot.

### How it works 🤔

In order to implement this feature, We have to make to bot sleep until `20 min` before the prayer's time. And then send a notification to the subscribed users.

Also when the time of the prayer's arrive we should also notify them.

So for that I use simple `channels` to make the bot sleep until the time of the prayer's time.

The first sleep is until the `20 min` before the prayer's time. And the second sleep is until the prayer's time.

for more details check the [notifier code](https://github.com/escalopa/gopray/tree/main/telegram/internal/adapters/notifier/notifier.go).

## Bot Options 🤖
  
| Command | Description |
| --- | --- |
| /help | Get help message showing all possible commands |
| /prayers | Get prayers time |
| /prayersdate | Get prayers time by date |
| /subscribe | Subscribe to get notified on the prayers time |
| /unsubscribe | Unsubscribe to not get notified on the prayers time |
| /lang | Change the bot language (Default English) |
| /feedback | Send feedback to the bot owner |
| /bug | Report a bug to the bot owner |

## References 📚 

- [Telegram API (Telego)](https://github.com/SakoDroid/telego)
- [Prayer Times Site](http://dumrt.ru/ru/help-info/prayertime/)


## Upcoming Features 🚀

- [x] Support date format for prayersdate command with leading zeros and with delimiters (. / -)
- [x] Implement subscriptions & notifications
- [x] Update text messages to be more user friendly
- [ ] Add different languages support (AR, RU)
