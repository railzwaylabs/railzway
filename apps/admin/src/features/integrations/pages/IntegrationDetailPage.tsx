import { Link, useParams, useNavigate } from "react-router-dom";
import { useIntegrationCatalog, useConnectIntegration, useIntegrationConnections } from "../hooks/useIntegrations";
import { ConnectionForm } from "../components/ConnectionForm";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Info, CheckCircle2 } from "lucide-react";
import { toast } from "sonner";
import { useMemo } from "react";

export default function IntegrationDetailPage() {
  const { orgId, integrationId } = useParams();
  const navigate = useNavigate();
  const { data: catalog, isLoading: isCatalogLoading } = useIntegrationCatalog();
  const { data: connections } = useIntegrationConnections();
  const { mutate: connect, isPending: isConnecting } = useConnectIntegration();

  const item = useMemo(() =>
    catalog?.find((i) => i.id === integrationId),
    [catalog, integrationId]
  );

  const existingConnection = useMemo(() =>
    connections?.find((c) => c.integration_id === integrationId),
    [connections, integrationId]
  );

  const handleConnect = (values: any) => {
    connect({
      integration_id: integrationId!,
      ...values,
    }, {
      onSuccess: () => {
        toast.success(`${item?.name} connected successfully`);
        navigate(`/orgs/${orgId}/integrations/connections`);
      },
      onError: (error: any) => {
        toast.error(`Failed to connect ${item?.name}: ${error.message}`);
      }
    });
  };

  if (isCatalogLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-10 w-64" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Skeleton className="h-64 md:col-span-1" />
          <Skeleton className="h-96 md:col-span-2" />
        </div>
      </div>
    );
  }

  if (!item) {
    return (
      <div className="text-center py-20">
        <h2 className="text-2xl font-bold">Integration not found</h2>
        <p className="text-muted-foreground mt-2">The integration you're looking for doesn't exist or is inactive.</p>
        <Link to={`/orgs/${orgId}/integrations`} className="text-primary hover:underline mt-4 block">
          Back to App Store
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/orgs/${orgId}/integrations`}>App Store</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{item.name}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="flex flex-col md:flex-row gap-6 items-start">
        <div className="w-full md:w-1/3 space-y-6">
          <Card>
            <CardHeader className="text-center">
              <div className="mx-auto w-20 h-20 rounded-xl bg-background border flex items-center justify-center p-2 mb-4 shadow-sm">
                {item.logo_url ? (
                  <img src={item.logo_url} alt={item.name} className="w-full h-full object-contain" />
                ) : (
                  <div className="text-3xl font-bold text-muted-foreground">{item.name[0]}</div>
                )}
              </div>
              <CardTitle className="text-2xl">{item.name}</CardTitle>
              <div className="flex justify-center mt-2">
                <Badge variant="secondary" className="capitalize">{item.type}</Badge>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground text-center leading-relaxed">
                {item.description}
              </p>
            </CardContent>
          </Card>

          <Alert>
            <Info className="h-4 w-4 shadow-sm" />
            <AlertTitle>Security Note</AlertTitle>
            <AlertDescription className="text-xs">
              Railzway uses enterprise-grade envelope encryption (AES-256) to secure your integration credentials.
              Secrets are never stored in plain text.
            </AlertDescription>
          </Alert>
        </div>

        <div className="flex-1 space-y-6">
          {existingConnection ? (
            <Card className="border-green-100 bg-green-50/30">
              <CardHeader>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-500" />
                  <CardTitle>Currently Connected</CardTitle>
                </div>
                <CardDescription>
                  This integration is active for your organization.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex justify-between items-center py-2 border-b">
                    <span className="text-sm font-medium">Connection Name</span>
                    <span className="text-sm">{existingConnection.name}</span>
                  </div>
                  <div className="flex justify-between items-center py-2 border-b">
                    <span className="text-sm font-medium">Status</span>
                    <Badge variant="default" className="bg-green-500">{existingConnection.status}</Badge>
                  </div>
                  <div className="pt-4">
                    <Button variant="outline" className="w-full" asChild>
                      <Link to={`/orgs/${orgId}/integrations/connections`}>
                        Manage Connection
                      </Link>
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardHeader>
                <CardTitle>Connect {item.name}</CardTitle>
                <CardDescription>
                  Fill out the form below to authorize Railzway to connect with your {item.name} account.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <ConnectionForm
                  item={item}
                  onSubmit={handleConnect}
                  isSubmitting={isConnecting}
                />
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
