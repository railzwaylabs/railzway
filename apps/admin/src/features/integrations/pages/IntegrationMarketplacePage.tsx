import { useNavigate, useSearchParams } from "react-router-dom";
import { useIntegrationCatalog, useIntegrationConnections } from "../hooks/useIntegrations";
import { IntegrationCard } from "../components/IntegrationCard";
import { Input } from "@/components/ui/input";
import { useEffect, useMemo } from "react";
import { Search } from "lucide-react";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";

const allowedTabs = [
  "all",
  "notification",
  "payment",
  "accounting",
  "crm",
  "data_warehouse",
  "analytics",
];

export default function IntegrationMarketplacePage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: catalog, isLoading: isCatalogLoading } = useIntegrationCatalog();
  const { data: connections } = useIntegrationConnections();

  const rawTab = searchParams.get("type");
  const activeTab = rawTab && allowedTabs.includes(rawTab) ? rawTab : "all";
  const search = searchParams.get("q") || "";

  useEffect(() => {
    if (!rawTab) return;
    if (rawTab === "all") {
      const params = new URLSearchParams(searchParams);
      params.delete("type");
      setSearchParams(params, { replace: true });
      return;
    }
    if (allowedTabs.includes(rawTab)) return;
    const params = new URLSearchParams(searchParams);
    params.delete("type");
    setSearchParams(params, { replace: true });
  }, [rawTab, searchParams, setSearchParams]);

  const setSearch = (val: string) => {
    const params = new URLSearchParams(searchParams);
    if (val) {
      params.set("q", val);
    } else {
      params.delete("q");
    }
    setSearchParams(params, { replace: true });
  };

  const setActiveTab = (val: string) => {
    const params = new URLSearchParams(searchParams);
    if (val && val !== "all") {
      params.set("type", val);
    } else {
      params.delete("type");
    }
    setSearchParams(params);
  };

  const filteredCatalog = useMemo(() => {
    if (!catalog) return [];
    return catalog.filter((item) => {
      const matchesSearch =
        item.name.toLowerCase().includes(search.toLowerCase()) ||
        item.description.toLowerCase().includes(search.toLowerCase());
      const matchesTab = activeTab === "all" || item.type === activeTab;
      return matchesSearch && matchesTab;
    });
  }, [catalog, search, activeTab]);

  const connectedIds = useMemo(() => {
    if (!connections) return new Set<string>();
    return new Set(connections.map((c) => c.integration_id));
  }, [connections]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-3xl font-bold tracking-tight">App Store</h1>
        <p className="text-muted-foreground">
          Connect your favorite tools to automate your billing workflow.
        </p>
      </div>

      <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full sm:w-auto">
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="notification">Notifications</TabsTrigger>
            <TabsTrigger value="payment">Payments</TabsTrigger>
            <TabsTrigger value="accounting">Accounting</TabsTrigger>
            <TabsTrigger value="crm">CRM</TabsTrigger>
            <TabsTrigger value="data_warehouse">Data Warehouse</TabsTrigger>
            <TabsTrigger value="analytics">Analytics</TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="relative w-full sm:w-72">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search integrations..."
            className="pl-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      {isCatalogLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {[...Array(8)].map((_, i) => (
            <Skeleton key={i} className="h-[220px] w-full rounded-xl" />
          ))}
        </div>
      ) : filteredCatalog.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-center border rounded-xl bg-muted/20">
          <div className="h-20 w-20 rounded-full bg-muted flex items-center justify-center mb-4">
            <Search className="h-10 w-10 text-muted-foreground/50" />
          </div>
          <h2 className="text-xl font-semibold">No integrations found</h2>
          <p className="text-muted-foreground max-w-xs mt-1">
            Try adjusting your search or filters to find what you're looking for.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {filteredCatalog.map((item) => (
            <IntegrationCard
              key={item.id}
              item={item}
              isConnected={connectedIds.has(item.id)}
              onConnect={() => navigate(item.id)}
              onView={() => navigate(item.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
