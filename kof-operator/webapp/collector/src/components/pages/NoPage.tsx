import { JSX } from "react";
import NotFound from "@/assets/not_found.png";

const NoPage = (): JSX.Element => {
  return (
    <div className="flex flex-col w-full h-screen justify-center items-center">
      <img className="h-[25%]" src={NotFound}></img>
      <h1 className="font-bold text-3xl">OOPS! PAGE NOT FOUND</h1>
    </div>
  );
};

export default NoPage;
