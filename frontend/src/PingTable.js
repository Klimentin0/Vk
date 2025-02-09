import React, { useState } from "react";

const PingTable = ({ pingResults }) => {
  const [currentPage, setCurrentPage] = useState(1);
  const rowsPerPage = 12;

  const indexOfLastRow = currentPage * rowsPerPage;
  const indexOfFirstRow = indexOfLastRow - rowsPerPage;
  const currentRows = pingResults.slice(indexOfFirstRow, indexOfLastRow);

  const handlePageChange = (newPage) => {
    setCurrentPage(newPage);
  };


  return (
    <div>
      <h3>Список всех пингов</h3>
      <table
        border="1"
        cellPadding="10"
        style={{
          width: "100%",
          borderCollapse: "collapse",
        }}
      >
        <thead>
          <tr>
            <th>ID контейнера</th>
            <th>Имя контейнера</th>
            <th>Время отклика</th>
            <th>Статус</th>
            <th>IP</th>
            <th>Дата пинга</th>
          </tr>
        </thead>
        <tbody>
          {currentRows.map((result, index) => {
            const timestamp = result.timestamp
              ? new Date(result.timestamp).toLocaleString()
              : "N/A";


            return (
              <tr
                key={index}
                
              >
                <td>{result.container_id}</td>
                <td>{result.container_name}</td>
                <td>{result.ping_duration.toFixed(3)}</td>
                <td style={{ color: result.status === "UP" ? "green" : "red" }}>
                  {result.status}
                </td>
                <td>{result.ip_address || "N/A"}</td>
                <td>{timestamp}</td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <div style={{ marginTop: "20px", textAlign: "center" }}>
        <button
          onClick={() => handlePageChange(currentPage - 1)}
          disabled={currentPage === 1}
        >
          Previous
        </button>
        <span style={{ margin: "0 10px" }}>Page {currentPage}</span>
        <button
          onClick={() => handlePageChange(currentPage + 1)}
          disabled={indexOfLastRow >= pingResults.length}
        >
          Next
        </button>
      </div>
    </div>
  );
};

export default PingTable;