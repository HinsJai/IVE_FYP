function get_records() {
  fetch(`/records_api`)
    .then((response) => response.json())
    .then((data) => {
      $("#myTable").DataTable({
        data: data[0].result,
        columns: [
          { data: "cameraID" },
          {
            data: "violation_type",
            render: function (data, _, _) {
              return data.join(", ");
            },
          },
          { data: "workplace" },
          {
            data: "time",
            render: function (data, _, _) {
              return moment(data)
                .tz("Africa/Abidjan")
                .format("MMM DD/MM/YYYY HH:mm:ss");
            },
          },
        ],
      });
    })
    .catch((error) => {
      console.error("Error fetching records:", error);
    });
}

window.onload = get_records;
