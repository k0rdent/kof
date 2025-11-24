import { JSX } from "react";
import { TableCell, TableRow } from "../generated/ui/table";
import { Alert, AlertDescription, AlertTitle } from "../generated/ui/alert";
import { CircleAlertIcon, TriangleAlertIcon } from "lucide-react";
import { MessageType } from "../pages/collectorPage/models";

const CustomResourceRow = ({
  name,
  messageType,
  message,
}: CustomResourceProps): JSX.Element => {
  return (
    <TableRow>
      <TableCell colSpan={7}>
        <h1 className="text-xl font-bold mb-2">{name}</h1>

        {messageType && message && (
          <Alert
            className={
              messageType === "error"
                ? "bg-destructive/10 text-destructive border-none"
                : "border-none bg-amber-600/10 text-amber-600"
            }
          >
            {messageType === "error" ? <TriangleAlertIcon /> : <CircleAlertIcon />}
            <AlertTitle>
              {messageType === "warning"
                ? "Configuration Warning"
                : messageType === "error"
                ? "Configuration Error"
                : undefined}
            </AlertTitle>
            <AlertDescription
              className={
                messageType === "error" ? "text-destructive/80" : "text-amber-600/80"
              }
            >
              {message}
            </AlertDescription>
          </Alert>
        )}
      </TableCell>
    </TableRow>
  );
};

export default CustomResourceRow;

interface CustomResourceProps {
  name: string;
  messageType?: MessageType;
  message?: string;
}
