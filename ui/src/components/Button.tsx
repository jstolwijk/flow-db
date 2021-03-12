import React from "react";
import { useHistory } from "react-router";

export const BackButton: React.FC = ({ children }) => {
  const history = useHistory();
  return <Button onClick={history.goBack}>{children}</Button>;
};

export const Button: React.FC<{ onClick: () => void }> = ({ children, onClick }) => {
  return (
    <button
      onClick={onClick}
      className="flex-shrink-0 bg-purple-600 text-white text-base font-semibold py-2 px-4 rounded-lg shadow-md hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 focus:ring-offset-purple-200"
    >
      {children}
    </button>
  );
};
