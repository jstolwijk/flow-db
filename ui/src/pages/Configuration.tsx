import React from "react";
import { useHistory } from "react-router";
import useSWR from "swr";
import { fetcher } from "../util/Fetcher";

const Configuration = () => {
  const { data, error } = useSWR("/api/configurations/current", fetcher);

  const history = useHistory();
  const handleRowClick = (streamName: string) => {
    history.push(`/configuration/data-streams/${streamName}`);
  };

  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;
  return (
    <div>
      <table className="table-auto">
        <thead>
          <tr>
            <th>Stream name</th>
            <th>Number of documents</th>
          </tr>
        </thead>
        <tbody>
          {data.map((dataStream: string, index: number) => (
            <tr className="bg-emerald-200" key={index} onClick={() => handleRowClick(dataStream)}>
              <td>{dataStream}</td>
              <td>0</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default Configuration;
