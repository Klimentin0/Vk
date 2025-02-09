import React, { useEffect, useState } from "react";
import axios from "axios";
import PingTable from "./PingTable";
import TopBar from "./TopBar";
import LatestUPTable from "./LatestUPTable";

function App() {
  const [pingResults, setPingResults] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Берём резултаты из API
  const fetchPingResults = async () => {
    try {
      const response = await axios.get(`${process.env.REACT_APP_API_URL}/ping-results/all`);
      console.log(response.data)
      setPingResults(response.data);
      setLoading(false);
    } catch (err) {
      setError("Failed to fetch ping results");
      setLoading(false);
    }
  };

  // ФФетчим с апдейтом каждый 10 сек
  useEffect(() => {
    fetchPingResults(); // Первый фетч
    const interval = setInterval(fetchPingResults, 10000); // апдейтим

    // анмаунт компонента
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>{error}</div>;
  }

  return (
    <div style={{ padding: "20px" }}>
      <TopBar />
      <LatestUPTable />
      <PingTable pingResults={pingResults} />
    </div>
  );
}

export default App;