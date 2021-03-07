import React, { useState } from "react";
import useSWR from "swr";
import DropDown from "../components/DropDown";
import { fetcher } from "../util/Fetcher";

enum SortDirection {
  ASC = "ASC",
  DESC = "DESC",
}

const DataStream = () => {
  const [sortDirection, setSortDirection] = useState(SortDirection.DESC);
  const { data, error } = useSWR("/api/data-streams/cars/recent?order=" + sortDirection, fetcher);

  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;
  return (
    <div>
      <DropDown
        id="sortDirectionSelector"
        name="sortDirection"
        options={[
          { value: SortDirection.DESC, title: "Descending" },
          { value: SortDirection.ASC, title: "Ascending" },
        ]}
        value={sortDirection}
        onValueSelected={(value) => setSortDirection(value as SortDirection)}
      />
      <ul>
        {data.map((dataStream: any) => (
          <li>
            <pre>{JSON.stringify(dataStream)}</pre>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default DataStream;
