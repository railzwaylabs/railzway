import i18n from "i18next"
import { initReactI18next } from "react-i18next"
import en from "../locales/en.json"
import id from "../locales/id.json"

const resources = {
  en: { translation: en },
  id: { translation: id }
}

const supportedLocales = new Set(Object.keys(resources))

function normalizeLanguage(raw?: string | null) {
  if (!raw) return "en"
  const lower = raw.toLowerCase()
  const base = lower.split(/[-_]/)[0]
  if (supportedLocales.has(base)) return base
  return "en"
}

function detectLanguage() {
  const stored = localStorage.getItem("rz_admin_locale")
  if (stored) return normalizeLanguage(stored)
  const nav = navigator.language?.toLowerCase() ?? "en"
  return normalizeLanguage(nav)
}

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: detectLanguage(),
    supportedLngs: Array.from(supportedLocales),
    nonExplicitSupportedLngs: true,
    fallbackLng: "en",
    interpolation: {
      escapeValue: false
    }
  })

export default i18n
