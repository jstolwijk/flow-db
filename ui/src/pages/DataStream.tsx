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
  const { data, mutate, isValidating } = useSWR(
    dataStreamName ? `/api/data-streams/${dataStreamName}/recent?order=${sortDirection}` : null,
    fetcher
  );

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
      <button
        type="button"
        className="focus:outline-none text-white text-sm py-2.5 px-5 rounded-md bg-blue-500 hover:bg-blue-600 hover:shadow-lg"
        onClick={() => mutate()}
        disabled={isValidating}
      >
        {isValidating ? "Loading..." : "Reload"}
      </button>
      {dataStreamName && <DataStreamDocuments data={data} />}
    </div>
  );
};

interface DataStreamSelectorProps {
  value: string | null;
  onValueSelected: (value: string) => void;
}

const DataStreamSelector: React.FC<DataStreamSelectorProps> = ({ value, onValueSelected }) => {
  const { data, error, mutate } = useSWR("/api/configurations/current", fetcher);

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
  data: any[];
}

const DataStreamDocuments: React.FC<DataStreamDocumentsProps> = ({ data }) => {
  return (
    <ul>
      <table className="table-auto">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>ID</th>
            <th>Document</th>
          </tr>
        </thead>
        <tbody>
          {data?.map((dataStream: any, index: number) => (
            <tr key={index}>
              <td>{dataStream._timestamp}</td>
              <td>{dataStream._id}</td>
              <td>
                <pre>{JSON.stringify(filterInternalFields(dataStream))}</pre>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </ul>
  );
};

const filterInternalFields = (json: any) => {
  return Object.keys(json)
    .filter((key) => !key.startsWith("_"))
    .reduce((acc: any, curr: string) => {
      return {
        ...acc,
        [curr]: json[curr],
      };
    }, {});
};

export default DataStream;
