import React from "react";

export interface DropDownProps {
  options: Option[];
  name: string;
  id: string;
  value: string | null;
  onValueSelected: (value: string) => void;
}

export interface Option {
  value: string;
  title: string;
}

const DropDown: React.FC<DropDownProps> = ({ options, name, id, onValueSelected, value }) => {
  return (
    <select
      name={name}
      id={id}
      value={value || options[0].value}
      onChange={(e) => onValueSelected(e.target.value)}
      className="w-full border bg-white rounded px-3 py-2 outline-none"
    >
      {options.map((option, index) => (
        <option key={index} value={option.value} className="py-1">
          {option.title}
        </option>
      ))}
    </select>
  );
};

export default DropDown;
