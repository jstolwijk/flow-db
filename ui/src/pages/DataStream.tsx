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
  const [dataStreamName, setDataStreamName] = useState<string | null>(null);

  return (
    <div>
      <DataStreamSelector value={dataStreamName} onValueSelected={setDataStreamName} />
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
      {dataStreamName && <DataStreamDocuments sortDirection={sortDirection} dataStreamName={dataStreamName} />}
    </div>
  );
};

interface DataStreamSelectorProps {
  value: string | null;
  onValueSelected: (value: string) => void;
}

const DataStreamSelector: React.FC<DataStreamSelectorProps> = ({ value, onValueSelected }) => {
  const { data, error } = useSWR("/api/configurations/current", fetcher);

  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;
  return (
    <DropDown
      id="DataStreamSelector"
      name="DataStreamSelector"
      options={data.map((dataStream: string) => ({ value: dataStream, title: dataStream }))}
      value={value}
      onValueSelected={onValueSelected}
    />
  );
};

interface DataStreamDocumentsProps {
  dataStreamName: string;
  sortDirection: SortDirection;
}

const DataStreamDocuments: React.FC<DataStreamDocumentsProps> = ({ dataStreamName, sortDirection }) => {
  const { data } = useSWR(
    dataStreamName ? `/api/data-streams/${dataStreamName}/recent?order=${sortDirection}` : null,
    fetcher
  );

  return (
    <ul>
      {data?.map((dataStream: any, index: number) => (
        <li key={index}>
          <pre>{JSON.stringify(dataStream)}</pre>
        </li>
      ))}
    </ul>
  );
};

export default DataStream;
