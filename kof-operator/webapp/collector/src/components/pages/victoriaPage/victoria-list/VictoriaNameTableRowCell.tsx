import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/generated/ui/hover-card";
import { TableCell } from "@/components/generated/ui/table";
import { JSX } from "react";
import { getVictoriaNameType } from "../utils";

const VictoriaNameTableRowCell = ({ name }: { name: string }): JSX.Element => {
  return (
    <TableCell>
      <HoverCard>
        <HoverCardTrigger className="flex flex-col">
          <span className="truncate">{name}</span>
          <span className="text-sm text-muted-foreground">
            {getVictoriaNameType(name)}
          </span>
        </HoverCardTrigger>
        <HoverCardContent className="w-fit">{name}</HoverCardContent>
      </HoverCard>
    </TableCell>
  );
};

export default VictoriaNameTableRowCell;
