import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { JSX, useEffect, useRef, useState } from "react";

interface SelectProps {
  placeholder: string;
  disabled: boolean;
  items: string[];
  callbackFn: (value: string) => void;
}

export default function SelectItems({
  placeholder,
  disabled,
  items,
  callbackFn,
}: SelectProps): JSX.Element {
  const [value, setValue] = useState<string>("");
  const initialized = useRef(false);

  const OnSelectChange = (value: string): void => {
    setValue(value);
    callbackFn(value);
  };

  useEffect(() => {
    if (!initialized.current && items.length > 0) {
      setValue(items[0]);
      callbackFn(items[0]);
      initialized.current = true;
    }
  }, [callbackFn, items]);

  return (
    <Select
      disabled={disabled}
      onValueChange={OnSelectChange}
      value={value}
      defaultValue={items[0]}
    >
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
