import { useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Check, ChevronsUpDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "./ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "./ui/command";
import { Label } from "./ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { cn } from "@/lib/utils";

export type AutoCompleteOption = {
  value: string;
  label?: string;
};

type AutoCompleteInputProps = {
  id: string;
  label: ReactNode;
  value: string;
  options: AutoCompleteOption[];
  placeholder?: string;
  onSearch?: (query: string) => Promise<AutoCompleteOption[]>;
  debounceMs?: number;
  minSearchLength?: number;
  onChange: (value: string) => void;
};

export default function AutoCompleteInput({
  id,
  label,
  value,
  options,
  placeholder,
  onSearch,
  debounceMs = 300,
  minSearchLength = 2,
  onChange
}: AutoCompleteInputProps) {
  const { t } = useTranslation();
  const [searchOptions, setSearchOptions] = useState(options);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setSearchOptions(options);
  }, [options]);

  const mergedOptions = useMemo(() => {
    const map = new Map<string, AutoCompleteOption>();
    for (const option of options) {
      map.set(option.value, option);
    }
    for (const option of searchOptions) {
      map.set(option.value, option);
    }
    return Array.from(map.values());
  }, [options, searchOptions]);

  const resolvedLabel = useMemo(() => {
    if (!value) return "";
    const match = mergedOptions.find((option) => option.value === value);
    return match?.label ?? value;
  }, [mergedOptions, value]);

  useEffect(() => {
    if (open) {
      setQuery("");
    }
  }, [open]);

  useEffect(() => {
    if (!onSearch) {
      return;
    }
    const trimmed = query.trim();
    if (!trimmed || trimmed.length < minSearchLength) {
      setSearchOptions(options);
      return;
    }

    let active = true;
    const handle = setTimeout(() => {
      setLoading(true);
      onSearch(query)
        .then((results) => {
          if (active) {
            setSearchOptions(results);
          }
        })
        .catch(() => {
          if (active) {
            setSearchOptions(options);
          }
        })
        .finally(() => {
          if (active) {
            setLoading(false);
          }
        });
    }, debounceMs);

    return () => {
      active = false;
      clearTimeout(handle);
    };
  }, [debounceMs, minSearchLength, onSearch, options, query]);

  const currentOptions = useMemo(() => {
    const base = onSearch ? searchOptions : options;
    if (!query.trim() || onSearch) return base;
    const lowered = query.trim().toLowerCase();
    return base.filter((option) => (option.label ?? option.value).toLowerCase().includes(lowered));
  }, [onSearch, options, query, searchOptions]);

  const selectPlaceholder = placeholder || t("common.select_placeholder");
  const searchPlaceholder = placeholder || t("common.search_placeholder");
  const emptyLabel = loading ? t("common.loading") : t("common.no_results");

  return (
    <div className="flag-label" style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
      <Label htmlFor={id}>{label}</Label>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            id={id}
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className="h-10 w-full justify-between border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
            data-testid={id}
          >
            <span className={cn("truncate text-left", !resolvedLabel && "text-muted-foreground")}>
              {resolvedLabel || selectPlaceholder}
            </span>
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-60" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-1" align="start">
          <Command shouldFilter={!onSearch}>
            <CommandInput
              placeholder={searchPlaceholder}
              value={query}
              onValueChange={setQuery}
            />
            <CommandList>
              <CommandEmpty>{emptyLabel}</CommandEmpty>
              <CommandGroup>
                {currentOptions.map((option) => (
                  <CommandItem
                    key={option.value}
                    value={option.label ?? option.value}
                    onSelect={() => {
                      onChange(option.value);
                      setOpen(false);
                    }}
                  >
                    <Check
                      className={cn(
                        "h-4 w-4",
                        value === option.value ? "opacity-100" : "opacity-0"
                      )}
                    />
                    <span className="truncate">{option.label ?? option.value}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}
