import React from "react";
import useSWR from "swr";
import { fetcher } from "../util/Fetcher";

const Configuration = () => {
  const { data, error } = useSWR("/api/configurations/current", fetcher);

  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;
  return (
    <div>
      <ul>
        {data.map((dataStream: string) => (
          <li>{dataStream}</li>
        ))}
      </ul>
    </div>
  );
};

export default Configuration;
