package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func makeChairChannelName(chairID string) string {
	return fmt.Sprintf("chair:%s", chairID)
}

func makeUserChannelName(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

type newRideNotificationPayload struct {
	RideID                string     `json:"ride_id"`
	PickupCoordinate      Coordinate `json:"pickup_coordinate"`
	DestinationCoordinate Coordinate `json:"destination_coordinate"`
	Fare                  int        `json:"fare"`
	Status                string     `json:"status"`
	CreatedAt             int64      `json:"created_at"`
	UpdateAt              int64      `json:"updated_at"`
}

func getNotificationInfo(ctx context.Context, tx *sqlx.Tx, user *User, ride *Ride) (*appGetNotificationResponseData, *chairGetNotificationResponseData, error) {
	yetSentRideStatus := RideStatus{}
	status := ""
	if err := tx.GetContext(ctx, &yetSentRideStatus, `SELECT * FROM ride_statuses WHERE ride_id = ? AND app_sent_at IS NULL ORDER BY created_at ASC LIMIT 1`, ride.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status, err = getLatestRideStatus(ctx, tx, ride.ID)
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	} else {
		status = yetSentRideStatus.Status
	}

	fare, err := calculateDiscountedFare(ctx, tx, user.ID, ride, ride.PickupLatitude, ride.PickupLongitude, ride.DestinationLatitude, ride.DestinationLongitude)
	if err != nil {
		return nil, nil, err
	}

	data := &appGetNotificationResponseData{
		RideID: ride.ID,
		PickupCoordinate: Coordinate{
			Latitude:  ride.PickupLatitude,
			Longitude: ride.PickupLongitude,
		},
		DestinationCoordinate: Coordinate{
			Latitude:  ride.DestinationLatitude,
			Longitude: ride.DestinationLongitude,
		},
		Fare:      fare,
		Status:    status,
		CreatedAt: ride.CreatedAt.UnixMilli(),
		UpdateAt:  ride.UpdatedAt.UnixMilli(),
	}
	var chairData *chairGetNotificationResponseData = nil

	if ride.ChairID.Valid {
		chair := &Chair{}
		if err := tx.GetContext(ctx, chair, `SELECT * FROM chairs WHERE id = ?`, ride.ChairID); err != nil {
			return nil, nil, err
		}

		stats, err := getChairStats(ctx, tx, chair.ID)
		if err != nil {
			return nil, nil, err
		}

		data.Chair = &appGetNotificationResponseChair{
			ID:    chair.ID,
			Name:  chair.Name,
			Model: chair.Model,
			Stats: stats,
		}
		chairData = &chairGetNotificationResponseData{
			RideID: ride.ID,
			User: simpleUser{
				ID:   user.ID,
				Name: fmt.Sprintf("%s %s", user.Firstname, user.Lastname),
			},
			PickupCoordinate: Coordinate{
				Latitude:  ride.PickupLatitude,
				Longitude: ride.PickupLongitude,
			},
			DestinationCoordinate: Coordinate{
				Latitude:  ride.DestinationLatitude,
				Longitude: ride.DestinationLongitude,
			},
			Status: status,
		}
	}

	if yetSentRideStatus.ID != "" {
		_, err := tx.ExecContext(ctx, `UPDATE ride_statuses SET app_sent_at = CURRENT_TIMESTAMP(6) WHERE id = ?`, yetSentRideStatus.ID)
		if err != nil {
			return nil, nil, err
		}
	}
	return data, chairData, nil
}

func publishRideUpdateNotification(ctx context.Context, user *User, appData *appGetNotificationResponseData, chairData *chairGetNotificationResponseData) error {
	buf, err := json.Marshal(appData)
	if err != nil {
		return err
	}
	if err := redisClient.Publish(ctx, makeUserChannelName(user.ID), buf).Err(); err != nil {
		return err
	}

	if chairData != nil {
		buf, err := json.Marshal(chairData)
		if err != nil {
			return err
		}
		if err := redisClient.Publish(ctx, makeChairChannelName(appData.Chair.ID), buf).Err(); err != nil {
			return err
		}
	}

	return nil
}

func publishNewRideNotification(ctx context.Context, userID string, payload *newRideNotificationPayload) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := redisClient.Publish(ctx, makeUserChannelName(userID), buf).Err(); err != nil {
		return err
	}

	return nil
}
