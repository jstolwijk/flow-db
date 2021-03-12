import React, { useMemo } from "react";
import { useParams } from "react-router";
import useSWR from "swr";
import { BackButton } from "../components/Button";
import { fetcher } from "../util/Fetcher";

interface Params {
  dataStreamName: string;
}

const DataStreamConfiguration = () => {
  const { dataStreamName } = useParams<Params>();

  const { data, error, isValidating } = useSWR(
    dataStreamName ? `/api/data-streams/${dataStreamName}/schema` : null,
    fetcher
  );

  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;
  return (
    <div>
      <BackButton>Back</BackButton>
      <div></div>
      <JSONTree json={data.properties} />
    </div>
  );
};

interface JSONTreeProps {
  json: any;
}

const JSONTree: React.FC<JSONTreeProps> = ({ json }) => {
  const keys = useMemo(() => Object.keys(json), [json]);

  return (
    <div>
      {keys.map((key) => (
        <div>{key}</div>
      ))}
    </div>
  );
};

export default DataStreamConfiguration;
