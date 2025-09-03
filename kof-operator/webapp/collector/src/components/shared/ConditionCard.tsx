import { JSX, ReactNode } from "react";
import { Card, CardContent, CardHeader } from "../generated/ui/card";
import { CircleAlert, CircleCheckBig } from "lucide-react";
import { Badge } from "../generated/ui/badge";
import { Condition } from "@/models/ObjectMeta";

export interface ConditionCardProps {
  condition: Condition;
  children?: ReactNode;
}

const ConditionCard = ({
  condition,
  children,
}: ConditionCardProps): JSX.Element => {
  const isHealthy: boolean = condition.isHealthy;

  return (
    <Card className="gap-2">
      <CardHeader className="flex flex-row justify-between items-center">
        <div className="flex gap-4">
          {isHealthy ? (
            <CircleCheckBig className="text-green-600 w-5 h-5" />
          ) : (
            <CircleAlert className="text-red-600 w-5 h-5" />
          )}
          <span className="font-medium">{condition.name}</span>
          {isHealthy ? (
            <Badge
              variant="default"
              className="bg-green-600 text-primary-foreground"
            >
              {condition.status}
            </Badge>
          ) : (
            <Badge variant="destructive">{condition.status}</Badge>
          )}
        </div>
        <div className="text-muted-foreground text-sm">
          {condition.modificationDate?.toLocaleString()}
        </div>
      </CardHeader>

      <CardContent>
        {condition.reason && (
          <div className="flex gap-2">
            <span>Reason: </span>
            <span>{condition.reason}</span>
          </div>
        )}
        {condition.message && (
          <div className="flex gap-2">
            <span>Message: </span>
            <span>{condition.message}</span>
          </div>
        )}
        {children}
      </CardContent>
    </Card>
  );
};

export default ConditionCard;
