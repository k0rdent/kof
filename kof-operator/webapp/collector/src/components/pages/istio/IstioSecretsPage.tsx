import { Button } from "@/components/generated/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from "@/components/generated/ui/table";
import Loader from "@/components/shared/Loader";
import { useIstio } from "@/providers/istio/IstioProvider";
import { BookLock } from "lucide-react";
import { JSX } from "react";
import CustomizedTableHead from "../collectorPage/components/collector-list/CollectorTableHead";
import { Separator } from "@/components/generated/ui/separator";

const IstioSecretsPage = (): JSX.Element => {
  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <header className="flex justify-between">
        <h1 className="font-bold text-3xl">Remote Secrets</h1>
      </header>
      <Separator />
      <IstioSecretsList />
    </div>
  );
};

const IstioSecretsList = (): JSX.Element => {
  const { fetch, data, isLoading, error } = useIstio();

  if (!isLoading && error) {
    return (
      <div className="flex flex-col justify-center items-center mt-[15%]">
        <span className="mb-3">
          Failed to fetch Istio secrets. Click "Reload" button to try again.
        </span>
        <Button className="cursor-pointer" onClick={() => fetch()}>
          Reload
        </Button>
      </div>
    );
  }

  if (isLoading && !data) {
    return <Loader />;
  }

  if (!isLoading && !data) {
    return (
      <div className="flex w-full h-screen justify-center items-center">
        No Secrets found
      </div>
    );
  }

  return (
    <Card className="w-full h-full gap-3">
      <CardHeader>
        <CardTitle>
          <div className="flex gap-4 items-center w-full h-full">
            <BookLock className="w-5 h-5" />
            <div className="flex gap-1 flex-col">
              <h1 className="text-lg font-bold">Remote Secrets</h1>
            </div>
          </div>
        </CardTitle>
      </CardHeader>

      <CardContent>
        <Table className="w-full table-fixed">
          <TableHeader>
            <TableRow className="font-bold">
              <CustomizedTableHead text={"Cluster Name"} width={70} />
              <CustomizedTableHead text={"Secret Name"} width={110} />
              <CustomizedTableHead text={"Secret Namespace"} width={120} />
              <CustomizedTableHead text={"Sync Status"} width={120} />
            </TableRow>
          </TableHeader>
          <TableBody>
            {data?.secrets.map((secret) => (
              <TableRow
                key={`${secret.clusterName}/${secret.namespace}/${secret.name}`}
              >
                <TableCell>{secret.clusterName}</TableCell>
                <TableCell>{secret.name}</TableCell>
                <TableCell>{secret.namespace}</TableCell>
                <TableCell>{secret.syncStatus}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default IstioSecretsPage;
