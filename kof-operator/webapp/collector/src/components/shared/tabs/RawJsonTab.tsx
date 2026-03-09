import JsonView from "@uiw/react-json-view";
import { lightTheme } from "@uiw/react-json-view/light";
import { githubDarkTheme } from "@uiw/react-json-view/githubDark";
import { JSX } from "react";
import { useTheme } from "@/providers/ThemeProvider";

interface CustomJsonViewProps {
  object: object;
  // Depth level for collapsing JSON.
  // Starts at 1 for the root object.
  // Use 0 to show JSON fully expanded (no collapse).
  depthLevel?: number;
  shortenTextAfterLength?: number;
}

const CustomJsonView = ({
  depthLevel = 2,
  object,
  shortenTextAfterLength,
}: CustomJsonViewProps): JSX.Element => {
  const { theme } = useTheme();
  return (
    <JsonView
      style={theme === "light" ? lightTheme : githubDarkTheme}
      value={object}
      displayDataTypes={false}
      className="w-full whitespace-normal wrap-break-word"
      shouldExpandNodeInitially={(_, props) => {
        if (depthLevel === 0) {
          return true;
        }
        // Expand nodes while their level is less than or equal to the configured depthLevel
        return props.level <= depthLevel;
      }}
      shortenTextAfterLength={shortenTextAfterLength}
    />
  );
};
export default CustomJsonView;
