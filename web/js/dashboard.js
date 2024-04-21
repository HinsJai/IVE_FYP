
const image = new Image();
window.onload = async function () {
  await get_warning_day_count();
  // await get_warning_month_record();
  await warning_record_filter();
  await get_warning_count_filter();
  $("#date-time-now").text(new Date().toLocaleString());
};

function daysInMonth(month, year) {
  return new Date(year, month, 0).getDate();
}

function get_month_days() {
  let date = new Date();
  let month = date.getMonth() + 1;
  let year = date.getFullYear();
  // console.log(month, year);
  return daysInMonth(month, year);
}

let warningData = new Array(12).fill(0);
let violationCount = new Array(3).fill(0);

async function get_warning_day_count() {
  const response = await fetch(`/get_warning_day_count_api`);
  const data = await response.json();
  $("#warning_day_count").text(data[0].result);
}

// async function warning_record_filter(duration) {

//   response = await fetch(`/warning_record_filter?duration=${duration}`)
//   if (response.status === 401) {
//     window.location.replace("/?unauth=true")
//     throw new Error("not login")
//   }
//   data = await response.json();
//   console.log(data);
// }

let chart = null


async function warning_record_filter(duration = "day") {
  violationCount = new Array(3).fill(0);
  $("#duration-text").text(`(${duration.toUpperCase()})`)

  try {
    const response = await fetch(`/warning_record_filter?duration=${duration}`)
    const data = await response.json();
    if (data[0].result.length > 0) {

      for (let i = 0; i < data[0].result.length; i++) {
        for (let j = 0; j < data[0].result[i].violation_type.length; j++) {
          if (data[0].result[i].violation_type[j] == "NO_SAFETY_VEST") {
            violationCount[0] += 1;
          } else if (data[0].result[i].violation_type[j] == "NO_MASK") {
            violationCount[1] += 1;
          } else if (data[0].result[i].violation_type[j] == "NO_HARDHAT") {
            violationCount[2] += 1;
          }
        }
      }
    } else {
      violationCount = new Array(3).fill(0);
    }

    if (chart == null) {
      chart = new Chart(document.getElementById("common violation"), {
        type: "doughnut",
        plugins: [
          {
            id: "customCanvasBackgroundImage",
            beforeDraw: (chart) => {
              if (image.complete) {
                const ctx = chart.ctx;
                const { top, left, width, height } = chart.chartArea;
                const x = left + width / 2 - image.width / 2;
                const y = top + height / 2 - image.height / 2;
                ctx.drawImage(image, x, y);
              } else {
                image.onload = () => chart.draw();
              }
            },
          },
        ],
        data: {
          labels: ["No Safety Vest", "No Mask", "No Hardhat"],
          datasets: [
            {
              label: "Violation Record",
              data: violationCount,
              backgroundColor: [
                "rgb(255, 99, 132)",
                "rgb(54, 162, 235)",
                "rgb(255, 205, 86)",
              ],

              hoverOffset: 4,
            },
          ],
        },
        options: {
          plugins: {
            legend: {
              labels: {
                color: 'white',
                font: {
                  size: 16
                }
              }
            },
            tooltip: {
              bodyFont: {
                size: 20,
                color: 'white'
              },
              titleFont: {
                size: 20,
                color: 'white'
              }
            }
          }
        },
      })
    }
    else {
      chart.data.labels = ["No Safety Vest", "No Mask", "No Hardhat"]
      chart.data.datasets = [
        {
          label: "Violation Record",
          data: violationCount,
          backgroundColor: [
            "rgb(255, 99, 132)",
            "rgb(54, 162, 235)",
            "rgb(255, 205, 86)",
          ],
          hoverOffset: 4,
        },
      ]
      chart.update();
      $("#duration-text").text(`(${duration.toUpperCase()})`)
    }
  } catch (error) {
    console.error("Error:", error);
  }
}

function get_warning_count_label(duration) {
  switch (duration) {
    case "hour":
      return ["00", "01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23"];
    case "day":
      const days = get_month_days();
      return Array.from({ length: days }, (_, index) => index + 1);;
    case "month":
      return [
        "Jan", "Feb", "Mar", "Apr", "May", "Jun",
        "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
      ];

  }
}

let warningChart = null


async function get_warning_count_filter(duration = "hour") {
  // results = await get_warning_year_count()
  //warningData = new Array(12).fill(0);
  response = await fetch(`/get_warning_count_filter_api?duration=${duration}`);
  let results = await response.json();

  let label = get_warning_count_label(duration);
  let key = ""
  console.log(results[0].result)
  if (results[0].result.length > 0) {
    let result = results[0].result
    // console.log(result)
    if ("month" in result[0]) {
      key = "month"
    }
    else if ("day" in result[0]) {
      key = "day"
    }
    else if ("hour" in result[0]) {
      key = "hour"
    }

    warningData = new Array(label.length).fill(0);
    // console.log(warningData)
    for (let i = 0; i < result.length; i++) {
      let index = result[i][key]
      if (Number.isNaN(index)) {
        console.log(result[i][key])
      }
      if (key == "day" || key == "month") {
        index = index - 1
      }
      warningData[index] = result[i]['count']

    }
  } else {
    warningData = new Array(label.length).fill(0);
  }

  if (warningChart == null) {
    warningChart = new Chart(document.getElementById("warning"), {
      type: "line",
      data: {
        labels: label,

        datasets: [
          {
            label: "Num of warning",
            data: warningData,
            backgroundColor: "yellow",
            borderWidth: 2,
            borderColor: "yellow",
          },
        ],
      },
      options: {
        responsive: true,
        scales: {
          x: {
            title: {
              display: true,
              text: `num of ${duration}`,
              color: 'white',
              font: {
                size: 20
              }
            },
            ticks: {
              font: {
                size: 20
              },
              color: 'white'
            }
          },
          y: {
            title: {
              display: true,
              text: `num of count`,
              color: 'white',
              font: {
                size: 20
              }
            },
            ticks: {
              font: {
                size: 20
              },
              color: 'white'
            }
          }
        },
        plugins: {
          legend: {
            labels: {
              font: {
                size: 20
              },
              color: 'white'
            }
          }
        }
      },

    });
  } else {

    warningChart.data.labels = label;
    warningChart.data.datasets = [
      {
        label: "Num of warning",
        data: warningData,
        backgroundColor: "yellow",
        borderWidth: 2,
        borderColor: "yellow",
      },
    ];
    warningChart.options.scales.x.title.text = `num of ${duration}`;
    warningChart.update();
    $("#warning-duration").text(`(${duration.toUpperCase()})`)
  }
}

setInterval(() => {
  warning_record_filter();
}, 15000);

setInterval(() => {
  get_warning_day_count();
}, 15000);

setInterval(() => {
  get_warning_count_filter();
}, 15000)