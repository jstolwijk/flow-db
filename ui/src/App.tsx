import React from "react";
import { BrowserRouter as Router, Switch, Route, Link } from "react-router-dom";
import NavBar from "./components/NavBar";
import Configuration from "./pages/Configuration";
import DataStream from "./pages/DataStream";
import DataStreamConfiguration from "./pages/DataStreamConfiguration";

export default function App() {
  return (
    <Router>
      <div>
        <NavBar />
        {/* <nav>
          <ul>
            <li>
              <Link to="/">Home</Link>
            </li>
            <li>
              <Link to="/configuration">Configuration</Link>
            </li>
            <li>
              <Link to="/data-streams">Data streams</Link>
            </li>
          </ul>
        </nav> */}
        <Switch>
          <Route path="/configuration/data-streams/:dataStreamName">
            <DataStreamConfiguration />
          </Route>
          <Route path="/configuration">
            <Configuration />
          </Route>
          <Route path="/data-streams">
            <DataStream />
          </Route>
          <Route path="/">
            <div>Home</div>
          </Route>
        </Switch>
      </div>
    </Router>
  );
}
