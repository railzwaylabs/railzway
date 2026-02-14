import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { CatalogItem } from "../types";
import { Button } from "@/components/ui/button";

interface IntegrationCardProps {
  item: CatalogItem;
  isConnected?: boolean;
  onConnect?: (item: CatalogItem) => void;
  onView?: (item: CatalogItem) => void;
}

export function IntegrationCard({ item, isConnected, onConnect, onView }: IntegrationCardProps) {
  return (
    <Card className="flex flex-col h-full hover:shadow-md transition-shadow cursor-pointer group" onClick={() => onView?.(item)}>
      <CardHeader className="flex flex-row items-center gap-4 space-y-0 pb-2">
        <div className="w-12 h-12 rounded-lg bg-background border flex items-center justify-center overflow-hidden p-1 shadow-sm">
          {item.logo_url ? (
            <img src={item.logo_url} alt={item.name} className="w-full h-full object-contain" />
          ) : (
            <div className="text-xl font-bold text-muted-foreground">{item.name[0]}</div>
          )}
        </div>
        <div className="flex-1 flex flex-col">
          <CardTitle className="text-lg leading-tight group-hover:text-primary transition-colors">{item.name}</CardTitle>
          <div className="flex items-center gap-2 mt-1">
            <Badge variant="secondary" className="capitalize text-[10px] px-1.5 py-0 h-4">
              {item.type}
            </Badge>
            {isConnected && (
              <Badge variant="default" className="text-[10px] px-1.5 py-0 h-4 bg-green-500 hover:bg-green-600">
                Connected
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex-1 pt-2">
        <CardDescription className="line-clamp-3 text-sm leading-relaxed">
          {item.description}
        </CardDescription>
      </CardContent>
      <div className="p-4 pt-0 mt-auto flex gap-2">
        <Button
          variant={isConnected ? "outline" : "default"}
          className="w-full text-xs h-8"
          onClick={(e) => {
            e.stopPropagation();
            onConnect?.(item);
          }}
        >
          {isConnected ? "Manage" : "Connect"}
        </Button>
      </div>
    </Card>
  );
}
