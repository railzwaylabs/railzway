import { Link, useParams } from "react-router-dom";
import { useIntegrationConnections, useDisconnectIntegration } from "../hooks/useIntegrations";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Trash2, ExternalLink, RefreshCw, AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { format } from "date-fns";

export default function MyConnectionsPage() {
  const { orgId } = useParams();
  const { data: connections, isLoading, isError } = useIntegrationConnections();
  const { mutate: disconnect, isPending: isDisconnecting } = useDisconnectIntegration();

  const handleDisconnect = (id: string, name: string) => {
    if (confirm(`Are you sure you want to disconnect ${name}?`)) {
      disconnect(id, {
        onSuccess: () => {
          toast.success(`Disconnected ${name}`);
        },
        onError: (err: any) => {
          toast.error(`Failed to disconnect: ${err.message}`);
        }
      });
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div className="flex flex-col gap-1">
          <h1 className="text-3xl font-bold tracking-tight">Active Connections</h1>
          <p className="text-muted-foreground">
            Manage your connected apps and services.
          </p>
        </div>
        <Button asChild>
          <Link to={`/orgs/${orgId}/integrations`}>
            Add New Integration
          </Link>
        </Button>
      </div>

      {isError ? (
        <Card className="border-destructive/20 bg-destructive/5">
          <CardContent className="flex flex-col items-center py-10">
            <AlertCircle className="h-10 w-10 text-destructive mb-4" />
            <h2 className="text-lg font-semibold">Failed to load connections</h2>
            <p className="text-muted-foreground">Please try again later or contact support.</p>
          </CardContent>
        </Card>
      ) : connections && connections.length > 0 ? (
        <Card>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Integration</TableHead>
                <TableHead>Connection Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Activity</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {connections.map((conn) => (
                <TableRow key={conn.id}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded border bg-background flex items-center justify-center p-1">
                        {conn.integration?.logo_url ? (
                          <img src={conn.integration.logo_url} alt="" className="w-full h-full object-contain" />
                        ) : (
                          <div className="text-xs font-bold">{conn.integration?.name?.[0] || 'I'}</div>
                        )}
                      </div>
                      <span className="font-medium">{conn.integration?.name || conn.integration_id}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm">
                    {conn.name}
                  </TableCell>
                  <TableCell>
                    <Badge variant={conn.status === 'active' ? 'default' : 'secondary'} className={conn.status === 'active' ? 'bg-green-500' : ''}>
                      {conn.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {conn.last_synced_at ? format(new Date(conn.last_synced_at), 'PPP p') : 'No activity'}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button variant="ghost" size="icon" asChild>
                        <Link to={`/orgs/${orgId}/integrations/${conn.integration_id}`}>
                          <ExternalLink className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:bg-destructive/10"
                        onClick={() => handleDisconnect(conn.id, conn.name)}
                        disabled={isDisconnecting}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      ) : (
        <div className="flex flex-col items-center justify-center py-24 text-center border rounded-xl border-dashed">
          <RefreshCw className="h-12 w-12 text-muted-foreground/30 mb-4" />
          <h2 className="text-xl font-semibold">No active connections</h2>
          <p className="text-muted-foreground max-w-xs mt-1">
            Browse the App Store to connect your first integration.
          </p>
          <Button variant="outline" className="mt-6" asChild>
            <Link to={`/orgs/${orgId}/integrations`}>Go to App Store</Link>
          </Button>
        </div>
      )}
    </div>
  );
}
