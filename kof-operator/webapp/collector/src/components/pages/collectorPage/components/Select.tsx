import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/generated/ui/select";
import { JSX } from "react";

interface SelectProps {
  placeholder: string;
  disabled: boolean;
  items: string[];
  value?: string;
  callbackFn: (value: string) => void;
}

export default function SelectItems({
  placeholder,
  disabled,
  items,
  callbackFn,
  value,
}: SelectProps): JSX.Element {
  const OnSelectChange = (value: string): void => {
    callbackFn(value);
  };

  return (
    <Select disabled={disabled} onValueChange={OnSelectChange} value={value}>
      <SelectTrigger className="w-[250px] [&>span]:truncate">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {items.map((item) => (
          <SelectItem key={item} value={item}>
            {item}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
