import React, { useEffect, useState } from "react";
import axios from "axios";

const LatestUPTable = () => {
  const [latestUPResults, setLatestUPResults] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchLatestUPResults = async () => {
    try {
      const response = await axios.get(`${process.env.REACT_APP_API_URL}/ping-results/latest-up-per-container`);
      setLatestUPResults(response.data);
      setLoading(false);
    } catch (err) {
      setError("Failed to fetch latest UP results per container");
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLatestUPResults(); 

    const interval = setInterval(fetchLatestUPResults, 5000); 

    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>{error}</div>;
  }

  return (
    <div>
      <h3>ПОСЛЕДНИЕ УСПЕШНЫЕ PING-запросы</h3>
      <table
        border="1"
        cellPadding="10"
        style={{
          width: "100%",
          borderCollapse: "collapse",
          marginBottom: "20px",
        }}
      >
        <thead>
          <tr>
            <th>Имя контейнера</th>
            <th>Время отклика</th>
            <th>IP</th>
            <th>Дата пинга</th>
          </tr>
        </thead>
        <tbody>
          {latestUPResults.length > 0 ? (
            latestUPResults.map((result, index) => {
              const timestamp = result.timestamp
                ? new Date(result.timestamp).toLocaleString()
                : "N/A";
              return (
                <tr key={index}>
                  <td>{result.container_name}</td>
                  <td>{result.ping_duration.toFixed(3)}</td>
                  <td>{result.ip_address || "N/A"}</td>
                  <td>{timestamp}</td>
                </tr>
              );
            })
          ) : (
            <tr>
              <td colSpan="3">No containers with UP status</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
};

export default LatestUPTable;