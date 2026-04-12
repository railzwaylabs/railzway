import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "./api";
import type { ReferenceCurrency } from "./types";
import { toast } from "../components/Toast";

type CurrencyCache = {
  data?: ReferenceCurrency[];
  promise?: Promise<ReferenceCurrency[]>;
};

const currencyCache: CurrencyCache = {};

export function useCurrencies() {
  const { t } = useTranslation();
  const [currencies, setCurrencies] = useState<ReferenceCurrency[]>(currencyCache.data ?? []);
  const [loading, setLoading] = useState(!currencyCache.data);

  useEffect(() => {
    if (currencyCache.data) {
      setLoading(false);
      return;
    }

    if (!currencyCache.promise) {
      currencyCache.promise = api.reference.currencies().then((data) => {
        currencyCache.data = data;
        return data;
      });
    }

    let active = true;
    currencyCache.promise
      .then((data) => {
        if (active) {
          setCurrencies(data);
        }
      })
      .catch((err) => {
        if (active) {
          toast.error(t("reference.toast.currencies_failed"), err instanceof Error ? err.message : undefined);
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [t]);

  const options = useMemo(
    () =>
      currencies.map((currency) => ({
        value: currency.code,
        label: `${currency.code} · ${currency.name}`,
      })),
    [currencies]
  );

  return { currencies, options, loading };
}
