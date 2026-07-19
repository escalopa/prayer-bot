# How Global Prayer Times calculates what it shows

This guide explains the user-facing calculations and Islamic information shown
by the global Telegram bot. It covers prayer times, high-latitude rules, Hijri
dates, Islamic occasions, Qibla direction, and the rolling calendar.

It is a transparent description of the bot, not a religious ruling. Local
mosques and recognized authorities may use different conventions, safety
margins, or moon-sighting decisions.

For the full equations, see the [public calculation site](https://escalopa.github.io/prayer-bot/)
or download its [PDF methodology](https://escalopa.github.io/prayer-bot/downloads/prayer-time-calculations.pdf).

## Location and local time

When a user shares a location, the bot:

1. Validates the latitude and longitude.
2. Resolves an IANA timezone and country.
3. Stores coordinates rounded to three decimal places, approximately a city
   block, instead of retaining the raw Telegram precision.
4. Uses the timezone's daylight-saving and historical offset rules when
   selecting the local day.

Google Maps helps resolve the timezone and approximate place only. It does not
provide prayer times, Hijri dates, Qibla direction, or Islamic occasions.

## Prayer times

The bot calculates solar events locally. Fajr and Isha depend on the selected
twilight convention, Asr depends on the selected madhab, and the other prayers
use sunrise, solar transit, and sunset.

| Prayer | Calculation |
| --- | --- |
| Fajr | The Sun reaches the selected negative Fajr angle before solar transit. |
| Sunrise | The calculated sunrise event. |
| Dhuhr | Solar transit, when the Sun crosses the local meridian. |
| Asr | A post-transit shadow-ratio elevation: factor 1 for Shafi'i, Maliki, and Hanbali; factor 2 for Hanafi. |
| Maghrib | The calculated sunset event. |
| Isha | The selected negative Isha angle, except Umm al-Qura, which uses 90 minutes after Maghrib. |

### Available calculation methods

| Method | Fajr | Isha | Initial regional default |
| --- | ---: | ---: | --- |
| Muslim World League | 18 degrees | 17 degrees | Countries without another mapping |
| Egyptian General Authority | 19.5 degrees | 17.5 degrees | Egypt |
| Umm al-Qura | 18.5 degrees | Maghrib + 90 minutes | Saudi Arabia |
| Karachi | 18 degrees | 18 degrees | Pakistan, India, Bangladesh, Afghanistan |
| ISNA | 15 degrees | 15 degrees | United States, Canada |
| Diyanet | 18 degrees | 17 degrees | Turkey |
| Kemenag | 20 degrees | 18 degrees | Indonesia |
| MUIS | 20 degrees | 18 degrees | Singapore |
| JAKIM | 20 degrees | 18 degrees | Malaysia |

The regional choice is only an initial default. A user can select another
method, madhab, high-latitude rule, and per-prayer minute correction.

## High-latitude rules

At some latitudes the Sun may not reach the required Fajr or Isha angle. If
sunrise and sunset still exist, the bot can estimate the missing event from the
length of the night:

- **Angle based:** uses the prayer angle divided by 60 as a fraction of night.
- **Middle of the night:** uses one half of the night.
- **One seventh:** uses the first or last seventh of the night.

These rules cannot create a complete schedule during polar day or polar night
when sunrise or sunset itself is absent. In that situation, users should follow
an appropriate local authority.

## Prayer corrections and rounding

Each displayed prayer can be corrected by -30 to +30 whole minutes. The bot
adds that correction to the calculated event and rounds the result to the
nearest minute. A correction for one prayer does not change the others.

## Qibla direction

The Qibla tool calculates the initial great-circle direction from the saved
rounded coordinates to the Kaaba, using:

- Kaaba latitude: 21.4225 degrees north;
- Kaaba longitude: 39.8262 degrees east;
- mean Earth radius for the displayed distance: 6,371.0088 kilometres.

For user latitude `phi`, longitude `lambda`, Kaaba latitude `phi_K`, and
longitude difference `Delta_lambda`, the initial bearing is:

```text
B = atan2(
      sin(Delta_lambda) cos(phi_K),
      cos(phi) sin(phi_K) - sin(phi) cos(phi_K) cos(Delta_lambda)
    )
```

The result is normalized to 0-360 degrees clockwise from geographic north:
north is 0 degrees, east is 90, south is 180, and west is 270.

This is an initial great-circle bearing, not a flat-map line. The live compass
rotates the arrow using the phone's absolute orientation sensor. Nearby metal,
magnetic interference, sensor calibration, and the difference between magnetic
and geographic north can affect the on-screen arrow. The numeric bearing
remains available when live orientation is unsupported.

## Gregorian and Hijri dates

The bot converts the user's local Gregorian day to a calculated Umm al-Qura
Hijri date. The bundled Umm al-Qura table covers 14 March 1937 through
16 November 2077. Outside that interval, the calendar library uses a
deterministic 30-year arithmetic Hijri cycle.

Users can select a correction from -2 to +2 days. If the local Gregorian day is
`G` and the correction is `a`, the bot converts `G + a`. Prayer times stay
attached to `G`; the correction changes only the Hijri label and which
Gregorian day matches an Islamic occasion.

Calculated calendars can differ from an official local crescent observation.
Eid, Ramadan, and other lunar dates should therefore be confirmed with the
relevant local authority.

## Islamic occasions

The bot does not store a separate Gregorian calendar of Islamic dates. For each
local Gregorian day, it calculates the corrected Hijri date and checks whether
the Hijri month and day match the curated catalog.

| Hijri date | Category | Occasion | Reference or qualification |
| --- | --- | --- | --- |
| 10 Muharram | Fasting opportunity | Ashura | [Sahih Muslim 1162a](https://sunnah.com/muslim:1162a) |
| 12 Rabi al-Awwal | Commonly observed | Mawlid al-Nabi | Exact historical date and observance differ; [Quran 33:56](https://quran.com/33/56), [Sahih Muslim 1162e](https://sunnah.com/muslim:1162e) |
| 27 Rajab | Commonly observed | Isra and Mi'raj | Precise calendar date is not established; [Quran 17:1](https://quran.com/17/1) |
| 15 Sha'ban | Commonly observed | Mid-Sha'ban | Practices and scholarly assessment of specific evidence differ |
| 1 Ramadan | Major occasion | Beginning of Ramadan | [Quran 2:185](https://quran.com/2/185) |
| 21 Ramadan | Major occasion | Beginning of the last ten nights | [Surah Al-Qadr](https://quran.com/al-qadr), [Sahih al-Bukhari 2017](https://sunnah.com/bukhari:2017) |
| 1 Shawwal | Major occasion | Eid al-Fitr | Local date should be confirmed; [Quran 2:185](https://quran.com/2/185) |
| 1 Dhu al-Hijjah | Major occasion | Beginning of the first ten days | [Sahih al-Bukhari 969](https://sunnah.com/bukhari:969) |
| 9 Dhu al-Hijjah | Fasting opportunity | Day of Arafah | Recommended fast applies to non-pilgrims; [Sahih Muslim 1162a](https://sunnah.com/muslim:1162a) |
| 10 Dhu al-Hijjah | Major occasion | Eid al-Adha | Local date should be confirmed; [Quran 22:36](https://quran.com/22/36) |

The labels **major**, **fasting opportunity**, and **commonly observed** are
deliberate. Commonly observed dates are not presented as universally agreed
dates or practices. Reminders for all three categories are disabled until a
user explicitly enables them.

## Rolling calendar

The private calendar feed is generated separately for each user from their
saved location, timezone, calculation method, madhab, high-latitude rule,
minute corrections, and Hijri correction.

Whenever a calendar provider fetches the private URL, the bot calculates:

- timed prayer events for today and the following 29 local days;
- all-day Islamic occasion events that fall in the same corrected Hijri window.

The feed rolls forward when fetched; it does not permanently store 30 days of
events or require a daily calendar job. Stable event identifiers let a calendar
provider update a prayer instead of creating a duplicate when settings change.
Google Calendar controls its own subscription refresh schedule, so a new day or
settings change may not appear immediately.

## Why another source may differ

Differences can come from:

- another Fajr/Isha convention or Asr factor;
- a different high-latitude policy;
- elevation, refraction, or solar-position assumptions;
- timezone or daylight-saving data;
- local safety margins and minute corrections;
- calculated Hijri dates versus local moon sighting;
- compass calibration or magnetic interference.

When a trusted local authority publishes a timetable or lunar-date decision for
the user's area, that local guidance should take precedence over a general
calculation.
