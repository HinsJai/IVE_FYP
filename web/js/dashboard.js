const image = new Image();
new Chart(document.getElementById("common violation"), {
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
        data: [300, 50, 100],
        backgroundColor: [
          "rgb(255, 99, 132)",
          "rgb(54, 162, 235)",
          "rgb(255, 205, 86)",
        ],
        hoverOffset: 4,
      },
    ],
  },
});

new Chart(document.getElementById("past 12 months"), {
  type: "line",
  data: {
    labels: [
      "Jan",
      "Feb",
      "Mar",
      "Apr",
      "May",
      "Jun",
      "Jul",
      "Aug",
      "Sep",
      "Oct",
      "Nov",
      "Dec",
    ],
    datasets: [
      {
        label: "Num of warning",
        data: [200, 156, 219, 129, 207, 153, 131, 28, 130, 142, 83, 68],
        backgroundColor: "yellow",
        borderWidth: 2,
        borderColor: "yellow",
      },
      {
        label: "Num of accident",
        data: [4, 1, 3, 8, 6, 7, 3, 3, 0, 2, 2, 3],
        backgroundColor: "red",
        borderWidth: 2,
        borderColor: "red",
      },
    ],
  },
  options: {
    responsive: true,
  },
});
